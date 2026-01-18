/**
 * Hotel Detail Page
 * Displays comprehensive hotel information with booking functionality
 */

import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Grid,
  Card,
  CardContent,
  Button,
  Chip,
  Rating,
  Skeleton,
  Alert,
  Divider,
  TextField,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
  Snackbar,
} from '@mui/material';
import {
  LocationOn as LocationIcon,
  Phone as PhoneIcon,
  Email as EmailIcon,
  AccessTime as TimeIcon,
  Wifi as WifiIcon,
  Pool as PoolIcon,
  Restaurant as RestaurantIcon,
  FitnessCenter as GymIcon,
  Spa as SpaIcon,
  LocalParking as ParkingIcon,
  MeetingRoom as RoomIcon,
  ArrowBack as ArrowBackIcon,
  CalendarMonth as CalendarIcon,
} from '@mui/icons-material';
import { hotelsService, reservationsService } from '../services';
import { useAuth } from '../context/AuthContext';
import { ROUTES, PLACEHOLDER_IMAGES, DEFAULT_TIMES } from '../constants';
import { formatPrice, getMinCheckInDate, getMinCheckOutDate, calculateNights, calculateTotalPrice } from '../utils/helpers';

const amenityDetails = {
  wifi: { icon: <WifiIcon />, label: 'Free WiFi' },
  pool: { icon: <PoolIcon />, label: 'Pool' },
  restaurant: { icon: <RestaurantIcon />, label: 'Restaurant' },
  gym: { icon: <GymIcon />, label: 'Gym' },
  spa: { icon: <SpaIcon />, label: 'Spa' },
  parking: { icon: <ParkingIcon />, label: 'Parking' },
};

const HotelDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [hotel, setHotel] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [bookingOpen, setBookingOpen] = useState(false);
  const [bookingLoading, setBookingLoading] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });
  const [checkIn, setCheckIn] = useState('');
  const [checkOut, setCheckOut] = useState('');

  useEffect(() => {
    const fetchHotel = async () => {
      try {
        setLoading(true);
        const response = await hotelsService.getById(id);
        setHotel(response);
      } catch (err) {
        console.error('Error fetching hotel:', err);
        setError('Could not load hotel information.');
      } finally {
        setLoading(false);
      }
    };

    if (id) {
      fetchHotel();
    }
  }, [id]);

  const handleBookingOpen = () => {
    if (!isAuthenticated) {
      navigate(ROUTES.LOGIN, { state: { from: { pathname: `/hotels/${id}` } } });
      return;
    }
    setBookingOpen(true);
  };

  const handleBookingClose = () => {
    setBookingOpen(false);
    setCheckIn('');
    setCheckOut('');
  };

  const handleBookingSubmit = async () => {
    if (!checkIn || !checkOut) {
      setSnackbar({ open: true, message: 'Please select check-in and check-out dates', severity: 'warning' });
      return;
    }

    if (new Date(checkIn) >= new Date(checkOut)) {
      setSnackbar({ open: true, message: 'Check-out date must be after check-in date', severity: 'warning' });
      return;
    }

    try {
      setBookingLoading(true);
      await reservationsService.create(hotel.id, hotel.name, checkIn, checkOut);
      setSnackbar({ open: true, message: 'Reservation created successfully!', severity: 'success' });
      handleBookingClose();
    } catch (err) {
      console.error('Error creating reservation:', err);
      setSnackbar({
        open: true,
        message: err.response?.data?.error || 'Error creating reservation',
        severity: 'error',
      });
    } finally {
      setBookingLoading(false);
    }
  };

  if (loading) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Skeleton variant="rectangular" height={400} sx={{ borderRadius: 2, mb: 4 }} />
        <Grid container spacing={4}>
          <Grid size={{ xs: 12, md: 8 }}>
            <Skeleton variant="text" height={60} width="60%" />
            <Skeleton variant="text" height={30} width="40%" />
            <Skeleton variant="text" height={200} />
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
            <Skeleton variant="rectangular" height={300} sx={{ borderRadius: 2 }} />
          </Grid>
        </Grid>
      </Container>
    );
  }

  if (error || !hotel) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Alert severity="error" sx={{ mb: 2 }}>
          {error || 'Hotel not found'}
        </Alert>
        <Button startIcon={<ArrowBackIcon />} onClick={() => navigate(ROUTES.SEARCH)}>
          Back to Search
        </Button>
      </Container>
    );
  }

  const mainImage = hotel.images?.[0] || PLACEHOLDER_IMAGES[0];
  const galleryImages = hotel.images?.slice(1, 4) || PLACEHOLDER_IMAGES.slice(1);
  const pricePerNight = hotel.price_per_night || hotel.pricePerNight || 0;
  const availableRooms = hotel.avaiable_rooms || hotel.avaiableRooms || 0;
  const checkInTime = hotel.check_in_time || hotel.checkInTime || DEFAULT_TIMES.CHECK_IN;
  const checkOutTime = hotel.check_out_time || hotel.checkOutTime || DEFAULT_TIMES.CHECK_OUT;

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Hero Image */}
      <Box
        sx={{
          height: { xs: 300, md: 450 },
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <Box
          component="img"
          src={mainImage}
          alt={hotel.name}
          sx={{
            width: '100%',
            height: '100%',
            objectFit: 'cover',
          }}
        />
        <Box
          sx={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            height: '50%',
            background: 'linear-gradient(to top, rgba(0,0,0,0.7) 0%, transparent 100%)',
          }}
        />
        <Container
          maxWidth="lg"
          sx={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            pb: 4,
          }}
        >
          <Button
            startIcon={<ArrowBackIcon />}
            onClick={() => navigate(-1)}
            sx={{ color: 'white', mb: 2 }}
          >
            Back
          </Button>
          <Typography variant="h2" sx={{ color: 'white', fontWeight: 700, mb: 1 }}>
            {hotel.name}
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', color: 'white' }}>
              <LocationIcon sx={{ mr: 0.5 }} />
              <Typography variant="body1">
                {hotel.city}, {hotel.state}, {hotel.country}
              </Typography>
            </Box>
            <Box sx={{ display: 'flex', alignItems: 'center' }}>
              <Rating value={hotel.rating || 0} precision={0.5} readOnly sx={{ mr: 1 }} />
              <Typography variant="body1" sx={{ color: 'white' }}>
                {hotel.rating?.toFixed(1) || '0.0'}
              </Typography>
            </Box>
          </Box>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Grid container spacing={4}>
          {/* Main Content */}
          <Grid size={{ xs: 12, md: 8 }}>
            {/* Gallery */}
            {galleryImages.length > 0 && (
              <Grid container spacing={2} sx={{ mb: 4 }}>
                {galleryImages.map((img, index) => (
                  <Grid key={index} size={{ xs: 4 }}>
                    <Box
                      component="img"
                      src={img}
                      alt={`${hotel.name} ${index + 2}`}
                      sx={{
                        width: '100%',
                        height: 150,
                        objectFit: 'cover',
                        borderRadius: 2,
                        cursor: 'pointer',
                        transition: 'transform 0.2s',
                        '&:hover': {
                          transform: 'scale(1.02)',
                        },
                      }}
                    />
                  </Grid>
                ))}
              </Grid>
            )}

            {/* Description */}
            <Card sx={{ mb: 4 }}>
              <CardContent sx={{ p: 4 }}>
                <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
                  Description
                </Typography>
                <Typography variant="body1" color="text.secondary" sx={{ lineHeight: 1.8 }}>
                  {hotel.description || 'Enjoy an unforgettable stay at this magnificent hotel. With world-class facilities and exceptional service, we guarantee a unique experience.'}
                </Typography>
              </CardContent>
            </Card>

            {/* Amenities */}
            {hotel.amenities && hotel.amenities.length > 0 && (
              <Card sx={{ mb: 4 }}>
                <CardContent sx={{ p: 4 }}>
                  <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
                    Amenities
                  </Typography>
                  <Grid container spacing={2}>
                    {hotel.amenities.map((amenity, index) => {
                      const detail = amenityDetails[amenity.toLowerCase()] || {
                        icon: <RoomIcon />,
                        label: amenity,
                      };
                      return (
                        <Grid key={index} size={{ xs: 6, sm: 4, md: 3 }}>
                          <Box
                            sx={{
                              display: 'flex',
                              alignItems: 'center',
                              gap: 1.5,
                              p: 2,
                              bgcolor: 'background.default',
                              borderRadius: 2,
                            }}
                          >
                            <Box sx={{ color: 'primary.main' }}>{detail.icon}</Box>
                            <Typography variant="body2">{detail.label}</Typography>
                          </Box>
                        </Grid>
                      );
                    })}
                  </Grid>
                </CardContent>
              </Card>
            )}

            {/* Contact Info */}
            <Card>
              <CardContent sx={{ p: 4 }}>
                <Typography variant="h5" sx={{ fontWeight: 600, mb: 3 }}>
                  Contact Information
                </Typography>
                <Grid container spacing={3}>
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                      <LocationIcon color="primary" />
                      <Box>
                        <Typography variant="caption" color="text.secondary">
                          Address
                        </Typography>
                        <Typography variant="body2">{hotel.address}</Typography>
                      </Box>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <PhoneIcon color="primary" />
                      <Box>
                        <Typography variant="caption" color="text.secondary">
                          Phone
                        </Typography>
                        <Typography variant="body2">{hotel.phone || 'Not available'}</Typography>
                      </Box>
                    </Box>
                  </Grid>
                  <Grid size={{ xs: 12, sm: 6 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                      <EmailIcon color="primary" />
                      <Box>
                        <Typography variant="caption" color="text.secondary">
                          Email
                        </Typography>
                        <Typography variant="body2">{hotel.email || 'Not available'}</Typography>
                      </Box>
                    </Box>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                      <TimeIcon color="primary" />
                      <Box>
                        <Typography variant="caption" color="text.secondary">
                          Hours
                        </Typography>
                        <Typography variant="body2">
                          Check-in: {checkInTime} | Check-out: {checkOutTime}
                        </Typography>
                      </Box>
                    </Box>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          </Grid>

          {/* Sidebar - Booking Card */}
          <Grid size={{ xs: 12, md: 4 }}>
            <Card sx={{ position: 'sticky', top: 100 }}>
              <CardContent sx={{ p: 4 }}>
                <Box sx={{ display: 'flex', alignItems: 'baseline', mb: 2 }}>
                  <Typography variant="h3" sx={{ fontWeight: 700, color: 'primary.main' }}>
                    {formatPrice(pricePerNight)}
                  </Typography>
                  <Typography variant="body1" color="text.secondary" sx={{ ml: 1 }}>
                    /night
                  </Typography>
                </Box>

                <Divider sx={{ my: 3 }} />

                <Box sx={{ mb: 3 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                    <RoomIcon color="primary" />
                    <Typography variant="body1">
                      <strong>{availableRooms}</strong> rooms available
                    </Typography>
                  </Box>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Rating value={hotel.rating || 0} precision={0.5} size="small" readOnly />
                    <Typography variant="body2" color="text.secondary">
                      ({hotel.rating?.toFixed(1) || '0.0'})
                    </Typography>
                  </Box>
                </Box>

                <Button
                  fullWidth
                  variant="contained"
                  size="large"
                  onClick={handleBookingOpen}
                  startIcon={<CalendarIcon />}
                  disabled={availableRooms === 0}
                  sx={{ py: 1.5 }}
                >
                  {availableRooms === 0 ? 'No Availability' : 'Book Now'}
                </Button>

                {!isAuthenticated && (
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    sx={{ display: 'block', textAlign: 'center', mt: 2 }}
                  >
                    Sign in required to make a reservation
                  </Typography>
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Container>

      {/* Booking Dialog */}
      <Dialog open={bookingOpen} onClose={handleBookingClose} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ pb: 1 }}>
          <Typography variant="h5" fontWeight={600}>
            Book {hotel?.name}
          </Typography>
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Select your stay dates
          </Typography>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                fullWidth
                label="Check-in Date"
                type="date"
                value={checkIn}
                onChange={(e) => setCheckIn(e.target.value)}
                InputLabelProps={{ shrink: true }}
                inputProps={{ min: getMinCheckInDate() }}
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                fullWidth
                label="Check-out Date"
                type="date"
                value={checkOut}
                onChange={(e) => setCheckOut(e.target.value)}
                InputLabelProps={{ shrink: true }}
                inputProps={{ min: getMinCheckOutDate(checkIn) }}
                disabled={!checkIn}
              />
            </Grid>
          </Grid>
          {checkIn && checkOut && (
            <Box sx={{ mt: 3, p: 2, bgcolor: 'background.default', borderRadius: 2 }}>
              <Typography variant="body2" color="text.secondary">
                Booking Summary:
              </Typography>
              <Typography variant="h6" sx={{ mt: 1 }}>
                {calculateNights(checkIn, checkOut)} night{calculateNights(checkIn, checkOut) !== 1 ? 's' : ''}
              </Typography>
              <Typography variant="h5" sx={{ color: 'primary.main', fontWeight: 600 }}>
                Total: {formatPrice(calculateTotalPrice(pricePerNight, checkIn, checkOut))}
              </Typography>
            </Box>
          )}
        </DialogContent>
        <DialogActions sx={{ p: 3, pt: 0 }}>
          <Button onClick={handleBookingClose} variant="outlined">
            Cancel
          </Button>
          <Button
            onClick={handleBookingSubmit}
            variant="contained"
            disabled={bookingLoading || !checkIn || !checkOut}
          >
            {bookingLoading ? <CircularProgress size={24} /> : 'Confirm Booking'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default HotelDetail;
