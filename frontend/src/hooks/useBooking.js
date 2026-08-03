/**
 * Estado del booking concierge (plan 13 fase 5, RV21/F13-05).
 * Fechas civiles locales, límites de rooms/guests, total en centavos espejo
 * del backend e Idempotency-Key POR INTENTO: reintentar el mismo payload
 * conserva la key (hotels-api dedupea); cambiarlo o confirmar rota la key.
 */

import { useRef, useState } from 'react';
import { useNavigate, useLocation } from 'react-router';
import { useAuth } from './useAuth';
import { useAvailability } from './queries';
import { useCreateReservation } from './mutations';
import { addDaysDateOnly, differenceInNights, isValidDateOnly, todayLocal } from '../utils/dateOnly';
import { previewTotalCents } from '../utils/money';
import { createAttemptTracker } from '../utils/idempotency';
import { BOOKING_LIMITS, ROUTES } from '../constants';

export const useBooking = (hotel) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated, user } = useAuth();

  const [checkIn, setCheckIn] = useState('');
  const [checkOut, setCheckOut] = useState('');
  const [numRooms, setNumRooms] = useState(1);
  const [numGuests, setNumGuests] = useState(1);
  const [confirmation, setConfirmation] = useState(null);

  const attemptTracker = useRef(createAttemptTracker());

  const minCheckIn = todayLocal();
  const minCheckOut = checkIn && isValidDateOnly(checkIn) ? addDaysDateOnly(checkIn, 1) : addDaysDateOnly(minCheckIn, 1);

  const datesValid =
    isValidDateOnly(checkIn) &&
    isValidDateOnly(checkOut) &&
    checkIn >= minCheckIn &&
    checkOut >= addDaysDateOnly(checkIn, 1);

  const nights = datesValid ? differenceInNights(checkIn, checkOut) : 0;
  const maxRooms = Math.max(1, Math.min(BOOKING_LIMITS.MAX_ROOMS, hotel?.available_rooms ?? 1));
  const totalCents = datesValid ? previewTotalCents(hotel?.price_per_night, nights, numRooms) : 0;

  // Feedback previo de cupo; POST /reservations sigue siendo la autoridad
  const availability = useAvailability(hotel?.id, datesValid ? checkIn : null, datesValid ? checkOut : null);

  const mutation = useCreateReservation(user?.id);

  const canSubmit =
    datesValid &&
    numRooms >= 1 &&
    numRooms <= maxRooms &&
    numGuests >= 1 &&
    numGuests <= BOOKING_LIMITS.MAX_GUESTS &&
    nights <= BOOKING_LIMITS.MAX_STAY_NIGHTS &&
    !mutation.isPending;

  const setCheckInSafe = (value) => {
    setCheckIn(value);
    // Si el checkout quedó antes del nuevo mínimo, arrastrarlo
    if (value && isValidDateOnly(value) && checkOut && checkOut <= value) {
      setCheckOut(addDaysDateOnly(value, 1));
    }
    mutation.reset();
  };

  const submit = async () => {
    if (!isAuthenticated) {
      navigate(ROUTES.LOGIN, { state: { from: location } });
      return;
    }
    if (!canSubmit) return;

    const request = {
      hotel_id: hotel.id,
      check_in: checkIn,
      check_out: checkOut,
      num_rooms: numRooms,
      num_guests: numGuests,
    };

    try {
      const created = await mutation.mutateAsync({
        request,
        idempotencyKey: attemptTracker.current.keyFor(request),
      });
      // Intento consumido: el próximo submit (otra reserva igual) es nuevo
      attemptTracker.current.reset();
      setConfirmation({ id: created.id, request, totalCents, nights });
    } catch (error) {
      // 409 de negocio (sin cupo) también consume el intento: reintentar tal
      // cual devolvería el replay del 409; el usuario debe cambiar fechas
      if (error?.code === 'no_availability') {
        attemptTracker.current.reset();
      }
      // request_in_flight / 5xx / red: conservar la key para el retry seguro
    }
  };

  return {
    checkIn,
    setCheckIn: setCheckInSafe,
    checkOut,
    setCheckOut: (value) => {
      setCheckOut(value);
      mutation.reset();
    },
    numRooms,
    setNumRooms,
    numGuests,
    setNumGuests,
    minCheckIn,
    minCheckOut,
    maxRooms,
    maxGuests: BOOKING_LIMITS.MAX_GUESTS,
    nights,
    totalCents,
    datesValid,
    availability,
    canSubmit,
    isAuthenticated,
    submit,
    submitError: mutation.error,
    isSubmitting: mutation.isPending,
    clearError: mutation.reset,
    confirmation,
    closeConfirmation: () => setConfirmation(null),
  };
};

export const bookingErrorCopy = (error) => {
  if (!error) return null;
  switch (error.code) {
    case 'no_availability':
      return {
        severity: 'warning',
        title: 'Those dates just sold out',
        message: 'Another guest took the last room for part of that range. Try different dates — nothing was charged.',
      };
    case 'request_in_flight':
      return {
        severity: 'info',
        title: 'Still processing your previous attempt',
        message: 'Give it a few seconds and try again — your booking is not duplicated.',
      };
    default:
      return {
        severity: 'error',
        title: 'The reservation could not be completed',
        message: error.message,
        traceId: error.traceId,
      };
  }
};
