/**
 * Query hooks (plan 13, F13-01): los componentes consumen estos hooks, nunca
 * el envelope ni Axios directo. El AbortSignal de TanStack Query llega a
 * Axios para cancelar búsquedas viejas (sin carreras de respuestas lentas).
 */

import { keepPreviousData, useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { hotelsService, authService, adminService, reservationsService, queryKeys } from '../../services';

/** Búsqueda global (Solr): pagina y ordena el índice completo. */
export const useHotelSearch = (params, options = {}) =>
  useQuery({
    queryKey: queryKeys.hotelSearch(params),
    queryFn: ({ signal }) => hotelsService.search(params, { signal }),
    // Paginar sin flashear el grid: la página anterior queda visible con un
    // indicador sutil hasta que llega la nueva
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    ...options,
  });

export const useHotel = (hotelId, options = {}) =>
  useQuery({
    queryKey: queryKeys.hotel(hotelId),
    queryFn: ({ signal }) => hotelsService.getById(hotelId, { signal }),
    enabled: !!hotelId,
    ...options,
  });

/** Disponibilidad como feedback previo; POST /reservations es la autoridad. */
export const useAvailability = (hotelId, checkIn, checkOut, options = {}) =>
  useQuery({
    queryKey: queryKeys.availability(hotelId, checkIn, checkOut),
    queryFn: async ({ signal }) => {
      const map = await hotelsService.checkAvailability([hotelId], checkIn, checkOut, { signal });
      return map[hotelId] ?? false;
    },
    enabled: !!(hotelId && checkIn && checkOut),
    staleTime: 10_000,
    ...options,
  });

/** Historial del usuario, canceladas incluidas (F13-06). */
export const useMyReservations = (userId, options = {}) =>
  useInfiniteQuery({
    queryKey: queryKeys.reservations(userId),
    initialPageParam: 0,
    queryFn: ({ signal, pageParam }) => reservationsService.listByUser(userId, { limit: 20, offset: pageParam }, { signal }),
    getNextPageParam: (lastPage, _pages, lastOffset) =>
      lastPage.items.length === 20 ? lastOffset + 20 : undefined,
    enabled: !!userId,
    ...options,
  });

/** Catálogo admin (hotels-api es fuente de verdad, sin lag del índice). */
export const useAdminHotels = (params, options = {}) =>
  useQuery({
    queryKey: queryKeys.adminHotels(params),
    queryFn: ({ signal }) => hotelsService.list(params, { signal }),
    placeholderData: keepPreviousData,
    ...options,
  });

export const useAdminUsers = (params, options = {}) =>
  useQuery({
    queryKey: queryKeys.adminUsers(params),
    queryFn: ({ signal }) => authService.listUsers(params, { signal }),
    placeholderData: keepPreviousData,
    ...options,
  });

/** Panel de servicios real/read-only (plan 11): refresco corto. */
export const useMicroservicesStatus = (options = {}) =>
  useQuery({
    queryKey: queryKeys.adminHealth(),
    queryFn: ({ signal }) => adminService.getMicroservicesStatus({ signal }),
    refetchInterval: 15_000,
    staleTime: 5_000,
    ...options,
  });
