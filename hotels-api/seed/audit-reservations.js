// Before upgrading an existing volume, stop hotels-api so this audit and the
// small idempotency migration see a stable database. This script never changes
// inventory or reservations. It refuses inconsistent data instead of guessing.
// Default: report only. Run with AUDIT_APPLY=1 to migrate proven completed keys.
// Example: docker compose exec -T mongo mongosh -u root -p "$MONGO_PASSWORD" \
//   --authenticationDatabase admin < hotels-api/seed/audit-reservations.js
// With apply: docker compose exec -T -e AUDIT_APPLY=1 mongo mongosh ...
const database = db.getSiblingDB(process.env.MONGO_DATABASE || 'hotels-api');
const crypto = require('crypto');
const errors = [];
const expected = new Map();
const hotels = new Map(database.hotels.find().toArray().map(h => [h._id.toString(), h]));
const reservations = new Map(database.reservations.find().toArray().map(r => [r._id.toString(), r]));
const day = d => new Date(d).toISOString().slice(0, 10);
for (const [id, r] of reservations) {
  if (!hotels.has(r.hotel_id)) errors.push(`reservation ${id}: missing hotel ${r.hotel_id}`);
  if (r.status === 'cancelled') continue;
  if (r.status !== 'confirmed' || !Number.isInteger(r.num_rooms) || r.num_rooms < 1 || !(r.check_out > r.check_in)) {
    errors.push(`reservation ${id}: invalid status/rooms/dates; inspect manually`);
    continue;
  }
  for (let date = new Date(r.check_in); date < r.check_out; date.setUTCDate(date.getUTCDate() + 1)) {
    const key = `${r.hotel_id}/${day(date)}`;
    expected.set(key, (expected.get(key) || 0) + r.num_rooms);
  }
}
for (const row of database.reservation_inventory.find()) {
  const key = `${row.hotel_id}/${row.date}`;
  const capacity = hotels.get(row.hotel_id)?.available_rooms;
  if (!Number.isInteger(row.booked) || row.booked < 0 || row.booked !== (expected.get(key) || 0) || capacity === undefined || row.booked > capacity) {
    errors.push(`inventory ${key}: booked=${row.booked}, expected=${expected.get(key) || 0}, capacity=${capacity}`);
  }
  expected.delete(key);
}
for (const [key, count] of expected) if (count > 0) errors.push(`inventory ${key}: missing; ${count} confirmed rooms`);
const updates = [];
for (const key of database.idempotency_keys.find({$or: [{payload_hash: {$exists: false}}, {reservation_id: {$exists: false}}]})) {
  if (!key.done || key.status !== 201) {
    errors.push(`idempotency ${key._id}: unproven legacy outcome; inspect associated request/reservation before deciding, do not blindly delete`);
    continue;
  }
  let id;
  try { id = JSON.parse(Buffer.from(key.body.buffer).toString('utf8')).data.id; }
  catch { errors.push(`idempotency ${key._id}: unreadable saved response`); continue; }
  const r = reservations.get(id);
  if (!r || r.user_id !== key.user_id) { errors.push(`idempotency ${key._id}: missing or mismatched reservation ${id}`); continue; }
  const canonical = {hotel_id:r.hotel_id,user_id:r.user_id,check_in:day(r.check_in),check_out:day(r.check_out),num_rooms:r.num_rooms,num_guests:r.num_guests};
  updates.push({id:key._id,reservation_id:id,payload_hash:crypto.createHash('sha256').update(JSON.stringify(canonical)).digest('hex')});
}
if (errors.length) { errors.forEach(print); throw new Error(`Audit failed: ${errors.length} inconsistencies. No changes made.`); }
print(`Audit OK: ${hotels.size} hotels, ${reservations.size} reservations, ${updates.length} proven legacy keys to migrate.`);
if (process.env.AUDIT_APPLY === '1') {
  for (const update of updates) database.idempotency_keys.updateOne({_id:update.id,payload_hash:{$exists:false}},{$set:{reservation_id:update.reservation_id,payload_hash:update.payload_hash}});
  print(`Idempotency migration applied; rerunning is safe. No reservation or inventory counters changed.`);
} else if (updates.length) print('Dry run only. Run with AUDIT_APPLY=1 after reviewing this result.');
