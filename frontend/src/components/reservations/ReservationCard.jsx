/**
 * Card de reserva (plan 13 fase 6): hotel, rango civil, noches, rooms,
 * guests, total del dominio (centavos + currency), status del backend e ID
 * corto copiable. Cancel solo cuando el lifecycle lo permite.
 */

import { Link } from 'react-router';
import { Box, Button, Card, CardContent, Chip, Typography } from '@mui/material';
import {
  CalendarMonth as CalendarIcon,
  MeetingRoom as RoomIcon,
  Person as PersonIcon,
} from '@mui/icons-material';
import { formatDateOnlyLabel, differenceInNights } from '../../utils/dateOnly';
import { formatMinorAmount } from '../../utils/money';
import { reservationDisplayStatus, canCancelReservation } from '../../utils/reservations';
import { prefetchHotelDetailChunk } from '../../utils/prefetch';
import { shortId } from '../../utils/helpers';

const ReservationCard = ({ reservation, onCancel }) => {
  const status = reservationDisplayStatus(reservation);
  const nights = differenceInNights(reservation.check_in, reservation.check_out);
  const cancelled = reservation.status === 'cancelled';

  return (
    <Card component="article" aria-label={`Reservation at ${reservation.hotel_name}`} sx={{ opacity: cancelled ? 0.75 : 1 }}>
      <CardContent sx={{ p: { xs: 2, sm: 3 } }}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: { xs: 'column', sm: 'row' },
            justifyContent: 'space-between',
            gap: 2,
          }}
        >
          <Box sx={{ minWidth: 0 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexWrap: 'wrap', mb: 1 }}>
              {/* h2 directo bajo el h1 de MyReservations (sin sección intermedia) */}
              <Typography variant="h6" component="h2" sx={{ fontWeight: 600 }}>
                {reservation.hotel_name}
              </Typography>
              <Chip label={status.label} color={status.color} size="small" />
            </Box>

            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.secondary', mb: 0.5 }}>
              <CalendarIcon sx={{ fontSize: 18 }} aria-hidden />
              <Typography variant="body2">
                {formatDateOnlyLabel(reservation.check_in)} → {formatDateOnlyLabel(reservation.check_out)}
                {' · '}
                {nights} night{nights === 1 ? '' : 's'}
              </Typography>
            </Box>

            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, color: 'text.secondary', flexWrap: 'wrap' }}>
              <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
                <RoomIcon sx={{ fontSize: 18 }} aria-hidden />
                <Typography variant="body2">
                  {reservation.num_rooms} room{reservation.num_rooms === 1 ? '' : 's'}
                </Typography>
              </Box>
              <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}>
                <PersonIcon sx={{ fontSize: 18 }} aria-hidden />
                <Typography variant="body2">
                  {reservation.num_guests} guest{reservation.num_guests === 1 ? '' : 's'}
                </Typography>
              </Box>
              <Typography variant="body2" sx={{ fontWeight: 600, color: 'text.primary' }}>
                {formatMinorAmount(reservation.total_price, reservation.currency)}
              </Typography>
            </Box>

            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
              ID:{' '}
              <Box component="span" sx={{ fontFamily: 'monospace', userSelect: 'all' }}>
                {shortId(reservation.id)}
              </Box>
            </Typography>
          </Box>

          <Box
            sx={{
              display: 'flex',
              flexDirection: { xs: 'row', sm: 'column' },
              gap: 1,
              alignItems: { xs: 'stretch', sm: 'flex-end' },
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <Button
              component={Link}
              to={`/hotels/${reservation.hotel_id}`}
              onPointerEnter={prefetchHotelDetailChunk}
              onFocus={prefetchHotelDetailChunk}
              variant="outlined"
              size="small"
            >
              View hotel
            </Button>
            {canCancelReservation(reservation) && (
              <Button variant="text" color="error" size="small" onClick={() => onCancel(reservation)}>
                Cancel
              </Button>
            )}
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
};

export default ReservationCard;
