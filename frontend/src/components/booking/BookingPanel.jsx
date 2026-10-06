/**
 * Booking concierge (plan 13 fase 5): panel sticky en desktop; en mobile un
 * action bar inferior fijo abre el dialog con el MISMO estado (useBooking
 * vive en HotelDetail — abrir el dialog no resetea lo elegido).
 */

import { useState } from 'react';
import { Box, Button, Card, CardContent, Dialog, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material';
import { Close as CloseIcon } from '@mui/icons-material';
import BookingForm from './BookingForm';
import { formatMajorAmount } from '../../utils/money';

export const BookingSidebar = ({ hotel, booking }) => (
  <Card sx={{ position: 'sticky', top: 96, display: { xs: 'none', md: 'block' } }}>
    <CardContent sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', alignItems: 'baseline', mb: 2 }}>
        <Typography variant="h4" component="p" sx={{ fontWeight: 700, color: 'primary.main' }}>
          {formatMajorAmount(hotel.price_per_night)}
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ ml: 1 }}>
          / night
        </Typography>
      </Box>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        {hotel.available_rooms} room{hotel.available_rooms === 1 ? '' : 's'} in this property
      </Typography>
      <BookingForm hotel={hotel} booking={booking} idPrefix="panel" />
    </CardContent>
  </Card>
);

export const BookingMobileBar = ({ hotel, booking }) => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Box
        sx={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          right: 0,
          zIndex: (theme) => theme.zIndex.appBar,
          display: { xs: 'flex', md: 'none' },
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 2,
          px: 2,
          py: 1.5,
          bgcolor: 'background.paper',
          borderTop: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box>
          <Typography variant="subtitle1" component="p" fontWeight={700}>
            {formatMajorAmount(hotel.price_per_night)}
            <Typography component="span" variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
              / night
            </Typography>
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {hotel.available_rooms} room{hotel.available_rooms === 1 ? '' : 's'} in this property
          </Typography>
        </Box>
        <Button variant="contained" size="large" onClick={() => setOpen(true)} sx={{ minWidth: 140 }}>
          Reserve
        </Button>
      </Box>

      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm" aria-labelledby="booking-dialog-title">
        <DialogTitle id="booking-dialog-title" sx={{ pr: 6 }}>
          Reserve {hotel.name}
          <IconButton
            aria-label="Close booking dialog"
            onClick={() => setOpen(false)}
            sx={{ position: 'absolute', right: 8, top: 10 }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <BookingForm hotel={hotel} booking={booking} idPrefix="dialog" />
        </DialogContent>
      </Dialog>
    </>
  );
};
