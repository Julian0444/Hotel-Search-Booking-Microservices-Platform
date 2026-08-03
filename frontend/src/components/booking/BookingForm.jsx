/**
 * Formulario del booking concierge (plan 13 fase 5): fechas civiles con
 * mínimos LOCALES, rooms/guests con límites, disponibilidad como feedback y
 * errores del dominio con copy propio. Compartido por el panel sticky
 * (desktop) y el dialog (mobile) — el estado vive en useBooking, arriba.
 */

import {
  Alert,
  AlertTitle,
  Box,
  Button,
  CircularProgress,
  MenuItem,
  TextField,
  Typography,
} from '@mui/material';
import BookingSummary from './BookingSummary';
import { bookingErrorCopy } from '../../hooks/useBooking';

const BookingForm = ({ hotel, booking, idPrefix = 'booking' }) => {
  const {
    checkIn,
    setCheckIn,
    checkOut,
    setCheckOut,
    numRooms,
    setNumRooms,
    numGuests,
    setNumGuests,
    minCheckIn,
    minCheckOut,
    maxRooms,
    maxGuests,
    nights,
    totalCents,
    datesValid,
    availability,
    canSubmit,
    isAuthenticated,
    submit,
    submitError,
    isSubmitting,
    clearError,
  } = booking;

  const errorCopy = bookingErrorCopy(submitError);

  return (
    <Box component="form" onSubmit={(event) => { event.preventDefault(); submit(); }} noValidate>
      <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1.5, mb: 2 }}>
        <TextField
          id={`${idPrefix}-check-in`}
          label="Check-in"
          type="date"
          value={checkIn}
          onChange={(event) => setCheckIn(event.target.value)}
          InputLabelProps={{ shrink: true }}
          inputProps={{ min: minCheckIn }}
          required
        />
        <TextField
          id={`${idPrefix}-check-out`}
          label="Check-out"
          type="date"
          value={checkOut}
          onChange={(event) => setCheckOut(event.target.value)}
          InputLabelProps={{ shrink: true }}
          inputProps={{ min: minCheckOut }}
          disabled={!checkIn}
          required
        />
        <TextField
          id={`${idPrefix}-rooms`}
          select
          label="Rooms"
          value={numRooms}
          onChange={(event) => setNumRooms(Number(event.target.value))}
        >
          {Array.from({ length: maxRooms }, (_, index) => index + 1).map((n) => (
            <MenuItem key={n} value={n}>
              {n}
            </MenuItem>
          ))}
        </TextField>
        <TextField
          id={`${idPrefix}-guests`}
          select
          label="Guests"
          value={numGuests}
          onChange={(event) => setNumGuests(Number(event.target.value))}
        >
          {Array.from({ length: maxGuests }, (_, index) => index + 1).map((n) => (
            <MenuItem key={n} value={n}>
              {n}
            </MenuItem>
          ))}
        </TextField>
      </Box>

      {datesValid && (
        <>
          {availability.data === false && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              No availability for those dates — try another range.
            </Alert>
          )}
          <Box sx={{ mb: 2, p: 2, bgcolor: 'background.default', borderRadius: 2 }}>
            <BookingSummary
              nightlyRate={hotel.price_per_night}
              nights={nights}
              numRooms={numRooms}
              numGuests={numGuests}
              totalCents={totalCents}
            />
          </Box>
        </>
      )}

      <Box aria-live="assertive">
        {errorCopy && (
          <Alert severity={errorCopy.severity} onClose={clearError} sx={{ mb: 2 }}>
            <AlertTitle>{errorCopy.title}</AlertTitle>
            {errorCopy.message}
            {errorCopy.traceId && (
              <Typography variant="caption" sx={{ display: 'block', mt: 1, fontFamily: 'monospace', userSelect: 'all' }}>
                Reference: {errorCopy.traceId}
              </Typography>
            )}
          </Alert>
        )}
      </Box>

      <Button
        type="submit"
        fullWidth
        variant="contained"
        size="large"
        disabled={isAuthenticated ? !canSubmit : false}
        sx={{ py: 1.4 }}
      >
        {isSubmitting ? (
          <CircularProgress size={24} color="inherit" aria-label="Confirming reservation" />
        ) : isAuthenticated ? (
          'Confirm reservation'
        ) : (
          'Sign in to reserve'
        )}
      </Button>

      {!isAuthenticated && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', textAlign: 'center', mt: 1 }}>
          You will come right back here after signing in.
        </Typography>
      )}
    </Box>
  );
};

export default BookingForm;
