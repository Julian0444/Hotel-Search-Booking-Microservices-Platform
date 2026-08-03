/**
 * Resumen del booking (plan 13 fase 5): noches, precio por noche, rooms,
 * guests y total — el total en CENTAVOS espejo del backend.
 */

import { Box, Divider, Typography } from '@mui/material';
import { formatMajorAmount, formatMinorAmount } from '../../utils/money';

const Row = ({ label, value, strong = false }) => (
  <Box sx={{ display: 'flex', justifyContent: 'space-between', py: 0.5 }}>
    <Typography variant={strong ? 'subtitle1' : 'body2'} color={strong ? 'text.primary' : 'text.secondary'}>
      {label}
    </Typography>
    <Typography variant={strong ? 'subtitle1' : 'body2'} fontWeight={strong ? 700 : 500}>
      {value}
    </Typography>
  </Box>
);

const BookingSummary = ({ nightlyRate, nights, numRooms, numGuests, totalCents, currency = 'USD' }) => (
  <Box aria-live="polite">
    <Row label={`${formatMajorAmount(nightlyRate, currency)} × ${nights} night${nights === 1 ? '' : 's'}`} value={`× ${numRooms} room${numRooms === 1 ? '' : 's'}`} />
    <Row label="Guests" value={String(numGuests)} />
    <Divider sx={{ my: 1 }} />
    <Row strong label="Total" value={formatMinorAmount(totalCents, currency)} />
  </Box>
);

export default BookingSummary;
