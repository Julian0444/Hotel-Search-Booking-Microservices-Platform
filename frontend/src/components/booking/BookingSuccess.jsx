/**
 * Confirmación del booking (plan 13 fase 5): evidencia (reservation ID
 * copiable), resumen y próximo paso — nunca un snackbar que desaparece.
 */

import { Link } from 'react-router';
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material';
import { CheckCircleOutline as CheckIcon } from '@mui/icons-material';
import BookingSummary from './BookingSummary';
import { formatDateOnlyLabel } from '../../utils/dateOnly';
import { shortId } from '../../utils/helpers';
import { ROUTES } from '../../constants';

const BookingSuccess = ({ hotel, confirmation, onClose }) => {
  if (!confirmation) return null;
  const { id, request, totalCents, nights } = confirmation;

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth="sm" aria-labelledby="booking-success-title">
      <DialogTitle id="booking-success-title" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <CheckIcon color="success" aria-hidden />
        Reservation confirmed
      </DialogTitle>
      <DialogContent>
        <Typography variant="body1" sx={{ mb: 1 }}>
          {hotel.name} — {formatDateOnlyLabel(request.check_in)} to {formatDateOnlyLabel(request.check_out)}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Confirmation ID:{' '}
          <Box component="span" sx={{ fontFamily: 'monospace', userSelect: 'all', fontWeight: 600 }}>
            {shortId(id)}
          </Box>
        </Typography>
        <Box sx={{ p: 2, bgcolor: 'background.default', borderRadius: 2 }}>
          <BookingSummary
            nightlyRate={hotel.price_per_night}
            nights={nights}
            numRooms={request.num_rooms}
            numGuests={request.num_guests}
            totalCents={totalCents}
          />
        </Box>
      </DialogContent>
      <DialogActions sx={{ p: 3, pt: 1 }}>
        <Button onClick={onClose} variant="outlined">
          Keep browsing
        </Button>
        <Button component={Link} to={ROUTES.RESERVATIONS} variant="contained">
          View my reservations
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default BookingSuccess;
