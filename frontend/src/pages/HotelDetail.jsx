/**
 * Hotel Detail (plan 13 fase 5, RV21/F13-05): gallery adaptativa, horas
 * "HH:mm" localizadas (nunca RFC3339), contacto solo si existe, amenities
 * normalizadas y el booking concierge — sticky en desktop, action bar en
 * mobile — con Idempotency-Key por intento.
 */

import { useNavigate, Link } from 'react-router';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Container,
  Grid,
  Rating,
  Skeleton,
  Typography,
} from '@mui/material';
import {
  LocationOn as LocationIcon,
  Phone as PhoneIcon,
  Email as EmailIcon,
  Schedule as ScheduleIcon,
  ArrowBack as ArrowBackIcon,
} from '@mui/icons-material';
import { useParams } from 'react-router';
import { useHotel } from '../hooks/queries';
import { useBooking } from '../hooks/useBooking';
import HotelGallery from '../components/Hotels/HotelGallery';
import { BookingSidebar, BookingMobileBar } from '../components/booking/BookingPanel';
import BookingSuccess from '../components/booking/BookingSuccess';
import { EmptyState, ErrorState, RouteMeta } from '../components/common';
import { amenityLabel } from '../utils/helpers';
import { formatTimeLabel } from '../utils/dateOnly';
import { ROUTES } from '../constants';

const DetailSkeleton = () => (
  <Container maxWidth="lg" sx={{ py: 4 }} aria-busy="true">
    <Skeleton variant="rectangular" sx={{ aspectRatio: '21 / 9', height: 'auto', borderRadius: 2, mb: 4 }} />
    <Grid container spacing={4}>
      <Grid size={{ xs: 12, md: 8 }}>
        <Skeleton variant="text" height={56} width="60%" />
        <Skeleton variant="text" height={28} width="40%" />
        <Skeleton variant="text" height={140} />
      </Grid>
      <Grid size={{ xs: 12, md: 4 }}>
        <Skeleton variant="rectangular" height={280} sx={{ borderRadius: 2 }} />
      </Grid>
    </Grid>
  </Container>
);

const HotelDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { data: hotel, isPending, isError, error, refetch } = useHotel(id);
  const booking = useBooking(hotel);

  if (isPending) {
    return <DetailSkeleton />;
  }

  if (isError) {
    if (error?.status === 404) {
      return (
        <Container maxWidth="lg" sx={{ py: 6 }}>
          <RouteMeta title="Stay not found" />
          <EmptyState
            title="That stay no longer exists"
            description="It may have been removed from the catalog."
            actionLabel="Back to search"
            onAction={() => navigate(ROUTES.SEARCH)}
          />
        </Container>
      );
    }
    return (
      <Container maxWidth="lg" sx={{ py: 6 }}>
        <RouteMeta title="Stay unavailable" />
        <ErrorState error={error} title="We could not load this stay" onRetry={refetch} />
      </Container>
    );
  }

  const rating = hotel.rating || 0;
  const hasContact = hotel.address || hotel.phone || hotel.email;

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh', pb: { xs: 10, md: 4 } }}>
      <RouteMeta
        title={`${hotel.name}, ${hotel.city}`}
        description={`${hotel.name} in ${hotel.city}, ${hotel.country} — from ${hotel.price_per_night} per night.`}
      />

      <Container maxWidth="lg" sx={{ pt: 3 }}>
        <Button component={Link} to={ROUTES.SEARCH} startIcon={<ArrowBackIcon />} sx={{ mb: 2 }}>
          Back to search
        </Button>

        <HotelGallery hotel={hotel} />

        <Grid container spacing={4} sx={{ mt: 0.5 }}>
          {/* Main content */}
          <Grid size={{ xs: 12, md: 8 }}>
            <Typography variant="h3" component="h1" sx={{ mb: 1 }}>
              {hotel.name}
            </Typography>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap', mb: 3 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', color: 'text.secondary' }}>
                <LocationIcon sx={{ mr: 0.5, fontSize: 20 }} aria-hidden />
                <Typography variant="body1">
                  {[hotel.city, hotel.state, hotel.country].filter(Boolean).join(', ')}
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Rating value={rating} precision={0.5} readOnly size="small" aria-hidden />
                <Typography variant="body2" color="text.secondary">
                  {rating.toFixed(1)} out of 5
                </Typography>
              </Box>
            </Box>

            {hotel.description && (
              <Card sx={{ mb: 3 }}>
                <CardContent sx={{ p: 3 }}>
                  <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>
                    About this stay
                  </Typography>
                  <Typography variant="body1" color="text.secondary" sx={{ lineHeight: 1.8 }}>
                    {hotel.description}
                  </Typography>
                </CardContent>
              </Card>
            )}

            <Card sx={{ mb: 3 }}>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>
                  Check-in and check-out
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <ScheduleIcon color="primary" aria-hidden />
                  <Typography variant="body1">
                    Check-in from <strong>{formatTimeLabel(hotel.check_in_time)}</strong> · check-out by{' '}
                    <strong>{formatTimeLabel(hotel.check_out_time)}</strong>
                  </Typography>
                </Box>
              </CardContent>
            </Card>

            {hotel.amenities?.length > 0 && (
              <Card sx={{ mb: 3 }}>
                <CardContent sx={{ p: 3 }}>
                  <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>
                    Amenities
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                    {hotel.amenities.map((amenity) => (
                      <Chip key={amenity} label={amenityLabel(amenity)} sx={{ bgcolor: 'background.default' }} />
                    ))}
                  </Box>
                </CardContent>
              </Card>
            )}

            {hasContact && (
              <Card>
                <CardContent sx={{ p: 3 }}>
                  <Typography variant="h6" component="h2" sx={{ mb: 1.5 }}>
                    Contact
                  </Typography>
                  <Box sx={{ display: 'grid', gap: 1.5 }}>
                    {hotel.address && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                        <LocationIcon color="primary" fontSize="small" aria-hidden />
                        <Typography variant="body2">{hotel.address}</Typography>
                      </Box>
                    )}
                    {hotel.phone && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                        <PhoneIcon color="primary" fontSize="small" aria-hidden />
                        <Typography variant="body2">{hotel.phone}</Typography>
                      </Box>
                    )}
                    {hotel.email && (
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                        <EmailIcon color="primary" fontSize="small" aria-hidden />
                        <Typography variant="body2">{hotel.email}</Typography>
                      </Box>
                    )}
                  </Box>
                </CardContent>
              </Card>
            )}
          </Grid>

          {/* Booking concierge (desktop sticky) */}
          <Grid size={{ xs: 12, md: 4 }}>
            <BookingSidebar hotel={hotel} booking={booking} />
          </Grid>
        </Grid>
      </Container>

      {/* Mobile action bar + dialog (mismo estado) */}
      <BookingMobileBar hotel={hotel} booking={booking} />

      <BookingSuccess hotel={hotel} confirmation={booking.confirmation} onClose={booking.closeConfirmation} />
    </Box>
  );
};

export default HotelDetail;
