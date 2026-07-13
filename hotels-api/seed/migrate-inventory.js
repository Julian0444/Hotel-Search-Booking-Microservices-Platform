// Backfill del plan 04: adapta reservas pre-existentes al modelo nuevo
// (status/num_rooms/created_at) y puebla reservation_inventory desde las
// reservas confirmadas. Correr UNA vez contra un volumen con datos viejos:
//
//   docker compose exec mongo mongosh -u root -p "$MONGO_PASSWORD" \
//     --authenticationDatabase admin hotels-api /seed/migrate-inventory.js
//
// (o montar/copiar este archivo al contenedor; con volumen nuevo no hace
// falta: no hay reservas viejas). Es idempotente: solo toca reservas sin
// status y regenera el inventario completo desde las confirmadas.

db = db.getSiblingDB('hotels-api');

// 1. Completar campos nuevos en reservas viejas (las que no tienen status)
const legacy = db.reservations.updateMany(
  { status: { $exists: false } },
  [
    {
      $set: {
        status: 'confirmed',
        num_rooms: 1,
        num_guests: { $ifNull: ['$num_guests', 1] },
        total_price: { $ifNull: ['$total_price', NumberLong('0')] },
        currency: { $ifNull: ['$currency', 'USD'] },
        created_at: { $ifNull: ['$created_at', '$$NOW'] },
      },
    },
  ]
);
print('backfill: ' + legacy.modifiedCount + ' reservas legacy actualizadas');

// 2. Regenerar el inventario desde las reservas confirmadas.
//    Noches: incluye check-in, excluye check-out (modelo canónico D4).
//    Capacity: join contra hotels por ObjectId(hotel_id).
db.reservation_inventory.deleteMany({});

const counters = {}; // "hotelID|YYYY-MM-DD" -> booked
const capacities = {}; // hotelID -> capacity

db.reservations.find({ status: 'confirmed' }).forEach(function (r) {
  if (!(r.check_in instanceof Date) || !(r.check_out instanceof Date)) {
    print('backfill: reserva ' + r._id + ' sin fechas válidas, salteada');
    return;
  }
  if (capacities[r.hotel_id] === undefined) {
    let hotel = null;
    try {
      hotel = db.hotels.findOne({ _id: ObjectId(r.hotel_id) });
    } catch (e) {
      // hotel_id no es un ObjectId válido
    }
    if (!hotel) {
      print('backfill: hotel ' + r.hotel_id + ' no encontrado, reserva ' + r._id + ' salteada');
      capacities[r.hotel_id] = null;
      return;
    }
    capacities[r.hotel_id] = hotel.avaiable_rooms;
  }
  if (capacities[r.hotel_id] === null) {
    return;
  }

  const rooms = r.num_rooms && r.num_rooms > 0 ? r.num_rooms : 1;
  const current = new Date(Date.UTC(r.check_in.getUTCFullYear(), r.check_in.getUTCMonth(), r.check_in.getUTCDate()));
  const end = new Date(Date.UTC(r.check_out.getUTCFullYear(), r.check_out.getUTCMonth(), r.check_out.getUTCDate()));
  while (current < end) {
    const day = current.toISOString().slice(0, 10);
    const key = r.hotel_id + '|' + day;
    counters[key] = (counters[key] || 0) + rooms;
    current.setUTCDate(current.getUTCDate() + 1);
  }
});

let inserted = 0;
Object.keys(counters).forEach(function (key) {
  const parts = key.split('|');
  db.reservation_inventory.insertOne({
    hotel_id: parts[0],
    date: parts[1],
    booked: counters[key],
    capacity: capacities[parts[0]],
  });
  inserted++;
});
print('backfill: ' + inserted + ' entradas de inventario generadas');

// Aviso si el backfill dejó noches sobrevendidas (reservas viejas > capacidad)
const overbooked = db.reservation_inventory.countDocuments({
  $expr: { $gt: ['$booked', '$capacity'] },
});
if (overbooked > 0) {
  print('backfill: ATENCIÓN — ' + overbooked + ' noches quedaron con booked > capacity (overbooking histórico)');
}
