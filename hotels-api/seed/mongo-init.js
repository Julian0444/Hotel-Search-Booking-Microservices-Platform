// Seed de hoteles demo (P7). Corre UNA sola vez, cuando el volumen de Mongo es
// nuevo (docker-entrypoint-initdb.d). Para re-sembrar: docker compose down -v.
//
// Los nombres de campo son los bson tags reales de hotels_dao.go — incluido el
// typo deliberado `avaiable_rooms` (se renombra recién en el plan 11 / C11).
//
// Nota (plan 06): estos hoteles NO aparecen en /search hasta que exista el
// backfill de Solr — el índice se alimenta por eventos de RabbitMQ que el seed
// no emite. Verificarlos vía GET /hotels/:id o mongosh.

db = db.getSiblingDB('hotels-api');

if (db.hotels.countDocuments() === 0) {
  db.hotels.insertMany([
    {
      name: 'Hotel Sierras de Córdoba',
      description: 'Hotel boutique frente a las sierras, con desayuno regional y pileta climatizada.',
      address: 'Av. San Martín 1200',
      city: 'Córdoba',
      state: 'Córdoba',
      country: 'Argentina',
      phone: '+54 351 555-0101',
      email: 'reservas@sierrascba.demo',
      price_per_night: 95.0,
      rating: 4.5,
      avaiable_rooms: 12,
      check_in_time: ISODate('2024-01-01T14:00:00Z'),
      check_out_time: ISODate('2024-01-01T10:00:00Z'),
      amenities: ['wifi', 'pileta', 'desayuno', 'estacionamiento'],
      images: [
        'https://images.unsplash.com/photo-1566073771259-6a8506099945?w=800',
        'https://images.unsplash.com/photo-1582719508461-905c673771fd?w=800',
      ],
    },
    {
      name: 'Palermo Soho Suites',
      description: 'Suites modernas en el corazón de Palermo, a pasos de bares y galerías.',
      address: 'Honduras 4800',
      city: 'Buenos Aires',
      state: 'CABA',
      country: 'Argentina',
      phone: '+54 11 555-0202',
      email: 'hola@palermosuites.demo',
      price_per_night: 140.0,
      rating: 4.7,
      avaiable_rooms: 20,
      check_in_time: ISODate('2024-01-01T15:00:00Z'),
      check_out_time: ISODate('2024-01-01T11:00:00Z'),
      amenities: ['wifi', 'gimnasio', 'rooftop', 'bar'],
      images: [
        'https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=800',
        'https://images.unsplash.com/photo-1590490360182-c33d57733427?w=800',
      ],
    },
    {
      name: 'Posada del Vino',
      description: 'Posada entre viñedos con cata incluida y vista a la cordillera.',
      address: 'Ruta 60 km 12',
      city: 'Mendoza',
      state: 'Mendoza',
      country: 'Argentina',
      phone: '+54 261 555-0303',
      email: 'info@posadadelvino.demo',
      price_per_night: 110.0,
      rating: 4.6,
      avaiable_rooms: 8,
      check_in_time: ISODate('2024-01-01T14:00:00Z'),
      check_out_time: ISODate('2024-01-01T10:30:00Z'),
      amenities: ['wifi', 'bodega', 'desayuno', 'spa'],
      images: [
        'https://images.unsplash.com/photo-1571896349842-33c89424de2d?w=800',
        'https://images.unsplash.com/photo-1584132967334-10e028bd69f7?w=800',
      ],
    },
    {
      name: 'Refugio del Lago',
      description: 'Cabañas de montaña sobre el Nahuel Huapi, chimenea y muelle privado.',
      address: 'Av. Bustillo km 8',
      city: 'Bariloche',
      state: 'Río Negro',
      country: 'Argentina',
      phone: '+54 294 555-0404',
      email: 'reservas@refugiodellago.demo',
      price_per_night: 180.0,
      rating: 4.8,
      avaiable_rooms: 6,
      check_in_time: ISODate('2024-01-01T15:00:00Z'),
      check_out_time: ISODate('2024-01-01T10:00:00Z'),
      amenities: ['wifi', 'chimenea', 'muelle', 'desayuno'],
      images: [
        'https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=800',
        'https://images.unsplash.com/photo-1445019980597-93fa8acb246c?w=800',
      ],
    },
    {
      name: 'Hostal de la Quebrada',
      description: 'Hostal colonial en el casco histórico, terraza con vista a los cerros.',
      address: 'Caseros 525',
      city: 'Salta',
      state: 'Salta',
      country: 'Argentina',
      phone: '+54 387 555-0505',
      email: 'contacto@hostalquebrada.demo',
      price_per_night: 70.0,
      rating: 4.3,
      avaiable_rooms: 15,
      check_in_time: ISODate('2024-01-01T13:00:00Z'),
      check_out_time: ISODate('2024-01-01T10:00:00Z'),
      amenities: ['wifi', 'terraza', 'desayuno'],
      images: [
        'https://images.unsplash.com/photo-1522798514-97ceb8c4f1c8?w=800',
        'https://images.unsplash.com/photo-1611892440504-42a792e24d32?w=800',
      ],
    },
  ]);
  print('seed: ' + db.hotels.countDocuments() + ' hoteles demo insertados');
} else {
  print('seed: la colección hotels ya tiene datos, no se siembra');
}

// Índices de reservas también acá: refuerza DB2 en un arranque limpio, antes
// de que hotels-api corra EnsureIndexes (misma spec, idempotente).
db.reservations.createIndex({ hotel_id: 1, check_in: 1, check_out: 1 });
db.reservations.createIndex({ user_id: 1 });
// Índice ÚNICO del inventario por hotel-noche: sostiene el claim atómico del
// no-overbooking (D1, plan 04). Misma spec que EnsureIndexes.
db.reservation_inventory.createIndex({ hotel_id: 1, date: 1 }, { unique: true });
print('seed: índices de reservations e inventario asegurados');
