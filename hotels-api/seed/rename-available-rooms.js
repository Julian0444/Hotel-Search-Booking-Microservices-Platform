// Migración one-off del rename C11 (plan 11): avaiable_rooms -> available_rooms.
//
// Solo hace falta sobre un volumen de Mongo EXISTENTE (sembrado antes del
// rename): con `docker compose down -v` el seed nuevo ya escribe el nombre
// correcto y esto no tiene nada que renombrar (updateMany con $rename de un
// campo ausente es un no-op, correr de más es inocuo).
//
// Uso (stack corriendo):
//   docker exec -i hotels-mongo mongosh -u root -p "$MONGO_PASSWORD" \
//     --authenticationDatabase admin < hotels-api/seed/rename-available-rooms.js
//
// Después del rename, reindexar Solr: POST /api/v1/reindex (admin) o
// docker compose down -v para recrear el core.
db = db.getSiblingDB('hotels-api');

const result = db.hotels.updateMany({}, { $rename: { avaiable_rooms: 'available_rooms' } });
print(`hotels migrados: matched=${result.matchedCount} modified=${result.modifiedCount}`);
