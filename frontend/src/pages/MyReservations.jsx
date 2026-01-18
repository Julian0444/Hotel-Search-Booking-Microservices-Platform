/**
 * My Reservations Page
 * Displays user's booking history with management options
 */

import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Card,
  CardContent,
  Grid,
  Button,
  Chip,
  Skeleton,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Snackbar,
  IconButton,
} from '@mui/material';
import {
  EventNote as EventNoteIcon,
  Hotel as HotelIcon,
  CalendarMonth as CalendarIcon,
  Delete as DeleteIcon,
  Visibility as ViewIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { reservationsService } from '../services';
import { useAuth } from '../context/AuthContext';
import { ROUTES } from '../constants';
import { formatDate, calculateNights, getReservationStatus } from '../utils/helpers';

const MyReservations = () => {
  const navigate = useNavigate();
  const { user, isAuthenticated } = useAuth();

  const [reservations, setReservations] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [deleteDialog, setDeleteDialog] = useState({ open: false, reservation: null });
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });

  useEffect(() => {
    if (!isAuthenticated) {
      navigate(ROUTES.LOGIN, { state: { from: { pathname: ROUTES.RESERVATIONS } } });
      return;
    }

    const fetchReservations = async () => {
      try {
        setLoading(true);
        const response = await reservationsService.getByUserId(user.id);
        setReservations(response || []);
      } catch (err) {
        console.error('Error fetching reservations:', err);
        setError('Could not load reservations');
      } finally {
        setLoading(false);
      }
    };

    fetchReservations();
  }, [isAuthenticated, user, navigate]);

  const handleDeleteClick = (reservation) => {
    setDeleteDialog({ open: true, reservation });
  };

  const handleDeleteConfirm = async () => {
    if (!deleteDialog.reservation) return;

    try {
      setDeleteLoading(true);
      await reservationsService.cancel(deleteDialog.reservation.id);
      setReservations(reservations.filter((r) => r.id !== deleteDialog.reservation.id));
      setSnackbar({ open: true, message: 'Reservation cancelled successfully', severity: 'success' });
    } catch (err) {
      console.error('Error canceling reservation:', err);
      setSnackbar({
        open: true,
        message: err.response?.data?.error || 'Error cancelling reservation',
        severity: 'error',
      });
    } finally {
      setDeleteLoading(false);
      setDeleteDialog({ open: false, reservation: null });
    }
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Header */}
      <Box
        sx={{
          bgcolor: 'primary.main',
          py: 6,
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            opacity: 0.1,
            backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='1'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
          }}
        />
        <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <EventNoteIcon sx={{ fontSize: 40, color: 'secondary.main', mr: 2 }} />
            <Box>
              <Typography variant="h3" sx={{ color: 'white', fontWeight: 600 }}>
                My Reservations
              </Typography>
              <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                Manage your hotel bookings
              </Typography>
            </Box>
          </Box>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ py: 4 }}>
        {error && (
          <Alert severity="error" sx={{ mb: 4 }}>
            {error}
          </Alert>
        )}

        {loading ? (
          <Grid container spacing={3}>
            {Array.from({ length: 3 }).map((_, index) => (
              <Grid key={index} size={{ xs: 12 }}>
                <Card>
                  <CardContent sx={{ p: 3 }}>
                    <Grid container spacing={2}>
                      <Grid size={{ xs: 12, sm: 8 }}>
                        <Skeleton variant="text" height={32} width="50%" />
                        <Skeleton variant="text" height={24} width="40%" />
                        <Skeleton variant="text" height={20} width="60%" />
                      </Grid>
                      <Grid size={{ xs: 12, sm: 4 }}>
                        <Skeleton variant="rectangular" height={40} />
                      </Grid>
                    </Grid>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        ) : reservations.length === 0 ? (
          <Box sx={{ textAlign: 'center', py: 8 }}>
            <HotelIcon sx={{ fontSize: 80, color: 'divider', mb: 2 }} />
            <Typography variant="h5" gutterBottom>
              No Reservations Yet
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
              Explore our hotels and make your first booking
            </Typography>
            <Button
              component={Link}
              to={ROUTES.SEARCH}
              variant="contained"
              size="large"
            >
              Search Hotels
            </Button>
          </Box>
        ) : (
          <Grid container spacing={3}>
            {reservations.map((reservation) => {
              const checkIn = reservation.check_in || reservation.checkIn;
              const checkOut = reservation.check_out || reservation.checkOut;
              const status = getReservationStatus(checkIn, checkOut);
              const nights = calculateNights(checkIn, checkOut);
              const hotelId = reservation.hotel_id || reservation.hotelId;
              const hotelName = reservation.hotel_name || reservation.hotelName || 'Hotel';

              return (
                <Grid key={reservation.id} size={{ xs: 12 }}>
                  <Card
                    sx={{
                      transition: 'all 0.3s ease',
                      '&:hover': {
                        transform: 'translateY(-2px)',
                      },
                    }}
                  >
                    <CardContent sx={{ p: { xs: 2, sm: 3 } }}>
                      <Grid container spacing={2} alignItems="center">
                        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                            <Box
                              sx={{
                                width: 56,
                                height: 56,
                                borderRadius: 2,
                                bgcolor: 'primary.main',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                              }}
                            >
                              <HotelIcon sx={{ color: 'white', fontSize: 28 }} />
                            </Box>
                            <Box>
                              <Typography variant="h6" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
                                {hotelName}
                              </Typography>
                              <Chip
                                label={status.label}
                                color={status.color}
                                size="small"
                                sx={{ mt: 0.5 }}
                              />
                            </Box>
                          </Box>
                        </Grid>

                        <Grid size={{ xs: 12, sm: 6, md: 4 }}>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                            <CalendarIcon sx={{ color: 'text.secondary', fontSize: 20 }} />
                            <Typography variant="body2" color="text.secondary">
                              {formatDate(checkIn)} - {formatDate(checkOut)}
                            </Typography>
                          </Box>
                          <Typography variant="body2" color="text.secondary">
                            {nights} night{nights !== 1 ? 's' : ''}
                          </Typography>
                        </Grid>

                        <Grid size={{ xs: 12, md: 4 }}>
                          <Box
                            sx={{
                              display: 'flex',
                              gap: 1,
                              justifyContent: { xs: 'flex-start', md: 'flex-end' },
                            }}
                          >
                            <Button
                              component={Link}
                              to={`/hotels/${hotelId}`}
                              variant="outlined"
                              size="small"
                              startIcon={<ViewIcon />}
                            >
                              View Hotel
                            </Button>
                            {status.canCancel && (
                              <IconButton
                                onClick={() => handleDeleteClick(reservation)}
                                color="error"
                                size="small"
                                sx={{
                                  border: '1px solid',
                                  borderColor: 'error.light',
                                }}
                              >
                                <DeleteIcon />
                              </IconButton>
                            )}
                          </Box>
                        </Grid>
                      </Grid>
                    </CardContent>
                  </Card>
                </Grid>
              );
            })}
          </Grid>
        )}
      </Container>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={deleteDialog.open}
        onClose={() => setDeleteDialog({ open: false, reservation: null })}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="warning" />
          Cancel Reservation
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            Are you sure you want to cancel your reservation at{' '}
            <strong>{deleteDialog.reservation?.hotel_name || deleteDialog.reservation?.hotelName}</strong>?
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button
            onClick={() => setDeleteDialog({ open: false, reservation: null })}
            variant="outlined"
          >
            Keep Reservation
          </Button>
          <Button
            onClick={handleDeleteConfirm}
            variant="contained"
            color="error"
            disabled={deleteLoading}
          >
            {deleteLoading ? 'Cancelling...' : 'Yes, Cancel'}
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

export default MyReservations;
