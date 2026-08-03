/**
 * Fixtures en el contrato FINAL de los planes 07/11 (plan 13 fase 0):
 * snake_case, `available_rooms` (sin typo), horas "HH:mm", fechas civiles
 * "YYYY-MM-DD", user_id string, total_price en centavos.
 * Nada de camelCase ni nombres legacy: si un componente los necesita para
 * renderizar, el componente está mal.
 */

// --- JWT de prueba: solo el payload importa (la SPA decodifica claims para
// validar exp/iss/aud; la firma la verifica el backend, no el browser) ---
const base64url = (obj) =>
  btoa(JSON.stringify(obj)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

export const makeJwt = (claims = {}) => {
  const now = Math.floor(Date.now() / 1000);
  const payload = {
    user_id: 7,
    username: 'julian',
    tipo: 'cliente',
    iss: 'users-api',
    aud: ['users-api', 'hotels-api', 'search-api'],
    iat: now,
    nbf: now,
    exp: now + 3600,
    ...claims,
  };
  return `${base64url({ alg: 'HS256', typ: 'JWT' })}.${base64url(payload)}.fake-signature`;
};

export const expiredJwt = () => makeJwt({ exp: Math.floor(Date.now() / 1000) - 60 });

// --- Hoteles ---
export const hotelFixture = {
  id: '64f1b2a3c4d5e6f7a8b9c0d1',
  name: 'Hotel Sierras de Córdoba',
  description: 'Editorial mountain retreat with spring water pools.',
  address: 'Av. San Martín 1200',
  city: 'Villa Carlos Paz',
  state: 'Córdoba',
  country: 'Argentina',
  phone: '+54 3541 420000',
  email: 'reservas@sierras.example',
  price_per_night: 150.5,
  rating: 4.5,
  available_rooms: 20,
  check_in_time: '15:00',
  check_out_time: '10:00',
  amenities: ['wifi', 'pool'],
  images: [
    'https://images.example/sierras-1.jpg',
    'https://images.example/sierras-2.jpg',
  ],
};

export const hotelFixtures = [
  hotelFixture,
  {
    ...hotelFixture,
    id: '64f1b2a3c4d5e6f7a8b9c0d2',
    name: 'Palacio Recoleta',
    city: 'Buenos Aires',
    state: 'CABA',
    price_per_night: 320,
    rating: 4.8,
    available_rooms: 8,
    amenities: ['wifi', 'spa', 'restaurant'],
    images: ['https://images.example/recoleta-1.jpg'],
  },
  {
    ...hotelFixture,
    id: '64f1b2a3c4d5e6f7a8b9c0d3',
    name: 'Posada del Glaciar',
    city: 'El Calafate',
    state: 'Santa Cruz',
    price_per_night: 95,
    rating: 4.1,
    available_rooms: 12,
    amenities: ['wifi', 'parking'],
    images: [],
  },
];

// --- Usuarios ---
export const clientUserFixture = { id: '7', username: 'julian', tipo: 'cliente' };
export const adminUserFixture = { id: '1', username: 'admin', tipo: 'administrador' };
export const userFixtures = [adminUserFixture, clientUserFixture];

export const loginResponseFixture = {
  user_id: '7',
  username: 'julian',
  token: makeJwt(),
  tipo: 'cliente',
};

// --- Reservas ---
export const reservationFixture = {
  id: 'a1b2c3d4e5f60718293a4b5c',
  hotel_id: hotelFixture.id,
  hotel_name: hotelFixture.name,
  user_id: '7',
  check_in: '2027-03-10',
  check_out: '2027-03-13',
  status: 'confirmed',
  num_rooms: 1,
  num_guests: 2,
  total_price: 45150, // 150.50 × 100 centavos × 3 noches × 1 room
  currency: 'USD',
  created_at: '2026-08-01T12:00:00Z',
};

export const cancelledReservationFixture = {
  ...reservationFixture,
  id: 'b2c3d4e5f60718293a4b5c6d',
  check_in: '2026-05-01',
  check_out: '2026-05-03',
  status: 'cancelled',
  total_price: 30100,
  cancelled_at: '2026-04-20T09:00:00Z',
};

export const completedReservationFixture = {
  ...reservationFixture,
  id: 'c3d4e5f60718293a4b5c6d7e',
  check_in: '2026-01-05',
  check_out: '2026-01-08',
  status: 'confirmed',
};

export const reservationFixtures = [
  reservationFixture,
  cancelledReservationFixture,
  completedReservationFixture,
];

// --- Panel de microservicios (shape real del plan 11) ---
export const microservicesFixture = {
  services: [
    {
      name: 'users-api',
      status: 'up',
      load_balanced: true,
      instances: [
        { name: 'users-api-1:8082', url: 'http://users-api-1:8082', status: 'up', latency_ms: 4 },
        { name: 'users-api-2:8082', url: 'http://users-api-2:8082', status: 'up', latency_ms: 6 },
        { name: 'users-api-3:8082', url: 'http://users-api-3:8082', status: 'up', latency_ms: 5 },
      ],
    },
    {
      name: 'hotels-api',
      status: 'up',
      load_balanced: false,
      instances: [
        { name: 'hotels-api:8081', url: 'http://hotels-api:8081', status: 'up', latency_ms: 3 },
      ],
    },
    {
      name: 'search-api',
      status: 'down',
      load_balanced: false,
      instances: [
        {
          name: 'search-api:8082',
          url: 'http://search-api:8082',
          status: 'down',
          latency_ms: 2001,
          error: 'context deadline exceeded',
        },
      ],
    },
  ],
  summary: { total_services: 3, total_instances: 5, healthy_services: 2 },
};

// --- Envelopes ---
export const listEnvelope = (items, { total = items.length, limit = 20, offset = 0 } = {}) => ({
  data: items,
  meta: { total, limit, offset },
});

export const errorEnvelope = (code, message, traceId = 'trace-test-123') => ({
  error: { code, message, trace_id: traceId },
});
