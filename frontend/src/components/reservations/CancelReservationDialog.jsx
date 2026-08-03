/**
 * Confirmación de cancelación (plan 13 fase 6): nombra el recurso, separa
 * keep/cancel y explica que la reserva queda en el historial.
 */

import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material';
import { formatDateOnlyLabel } from '../../utils/dateOnly';

const CancelReservationDialog = ({ reservation, isCancelling, error, onConfirm, onClose }) => (
  <Dialog open={!!reservation} onClose={onClose} maxWidth="sm" fullWidth aria-labelledby="cancel-reservation-title">
    <DialogTitle id="cancel-reservation-title">Cancel this reservation?</DialogTitle>
    <DialogContent>
      {reservation && (
        <Typography variant="body1" sx={{ mb: 1 }}>
          <strong>{reservation.hotel_name}</strong>, {formatDateOnlyLabel(reservation.check_in)} →{' '}
          {formatDateOnlyLabel(reservation.check_out)}
        </Typography>
      )}
      <Typography variant="body2" color="text.secondary">
        The stay will be released for other guests. The reservation stays in your history as cancelled.
      </Typography>
      {error && (
        <Alert severity="error" sx={{ mt: 2 }}>
          {error.message}
        </Alert>
      )}
    </DialogContent>
    <DialogActions sx={{ p: 2 }}>
      <Button onClick={onClose} variant="outlined" disabled={isCancelling}>
        Keep reservation
      </Button>
      <Button onClick={onConfirm} variant="contained" color="error" disabled={isCancelling}>
        {isCancelling ? 'Cancelling…' : 'Cancel reservation'}
      </Button>
    </DialogActions>
  </Dialog>
);

export default CancelReservationDialog;
