/**
 * Mutation hooks (plan 13): invalidación de queries en un solo lugar.
 * Sin retry automático — la idempotencia del booking la maneja el caller
 * conservando su Idempotency-Key por intento (F13-05).
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { adminService, authService, reservationsService, queryKeys } from '../../services';

/** Alta de reserva; invalida disponibilidad, hotel y el historial del user. */
export const useCreateReservation = (userId) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ request, idempotencyKey }) =>
      reservationsService.create(request, { idempotencyKey }),
    onSuccess: (_created, { request }) => {
      queryClient.invalidateQueries({ queryKey: ['availability', request.hotel_id] });
      queryClient.invalidateQueries({ queryKey: queryKeys.hotel(request.hotel_id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.reservations(userId) });
    },
  });
};

/** Cancelación: invalida el historial (la card pasa a Cancelled, no se borra). */
export const useCancelReservation = (userId) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (reservationId) => reservationsService.cancel(reservationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.reservations(userId) });
      queryClient.invalidateQueries({ queryKey: ['availability'] });
    },
  });
};

export const useSaveHotel = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ hotelId, data }) =>
      hotelId ? adminService.updateHotel(hotelId, data) : adminService.createHotel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'hotels'] });
      queryClient.invalidateQueries({ queryKey: ['hotels', 'search'] });
      queryClient.invalidateQueries({ queryKey: ['hotel'] });
      queryClient.invalidateQueries({ queryKey: ['availability'] });
    },
  });
};

export const useDeleteHotel = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (hotelId) => adminService.deleteHotel(hotelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'hotels'] });
      queryClient.invalidateQueries({ queryKey: ['hotels', 'search'] });
    },
  });
};

export const useDeleteUser = () => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId) => authService.deleteUser(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
    },
  });
};
