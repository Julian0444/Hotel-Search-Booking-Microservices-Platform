/**
 * QueryClient con defaults deliberados (plan 13, F13-01) y las query keys de
 * toda la app en un solo lugar — invalidar por prefijo depende de esto.
 */

import { QueryClient } from '@tanstack/react-query';

export const queryKeys = {
  hotelSearch: (params) => ['hotels', 'search', params],
  hotel: (id) => ['hotel', id],
  availability: (hotelId, checkIn, checkOut) => ['availability', hotelId, checkIn, checkOut],
  reservations: (userId) => ['reservations', userId],
  adminHotels: (params) => ['admin', 'hotels', params],
  adminUsers: (params) => ['admin', 'users', params],
  adminHealth: () => ['admin', 'health'],
};

export const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        // Reintento: una sola vez y nunca sobre 4xx (un 401/404 repetido no
        // va a mejorar); las mutations no reintentan solas (idempotencia la
        // maneja el caller con su Idempotency-Key)
        retry: (failureCount, error) => !(error?.isClientError) && failureCount < 1,
        staleTime: 30_000,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: false,
      },
    },
  });
