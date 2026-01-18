/**
 * Hotel Form Page
 * Create or edit hotel form for administrators
 */

import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Card,
  CardContent,
  TextField,
  Button,
  Grid,
  Alert,
  CircularProgress,
  Chip,
  IconButton,
  InputAdornment,
  Snackbar,
} from '@mui/material';
import {
  Save as SaveIcon,
  ArrowBack as ArrowBackIcon,
  Add as AddIcon,
  Close as CloseIcon,
  Hotel as HotelIcon,
} from '@mui/icons-material';
import { useForm } from 'react-hook-form';
import { hotelsService, adminService } from '../../services';
import { useAuth } from '../../context/AuthContext';
import { ROUTES, DEFAULT_TIMES, VALIDATION } from '../../constants';

const HotelForm = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const { isAdmin, isAuthenticated } = useAuth();
  const isEditing = !!id;

  const [loading, setLoading] = useState(false);
  const [fetchLoading, setFetchLoading] = useState(isEditing);
  const [error, setError] = useState(null);
  const [snackbar, setSnackbar] = useState({ open: false, message: '', severity: 'success' });
  const [amenityInput, setAmenityInput] = useState('');
  const [imageInput, setImageInput] = useState('');

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm({
    defaultValues: {
      name: '',
      description: '',
      address: '',
      city: '',
      state: '',
      country: '',
      phone: '',
      email: '',
      price_per_night: '',
      rating: '',
      avaiable_rooms: '',
      check_in_time: DEFAULT_TIMES.CHECK_IN,
      check_out_time: DEFAULT_TIMES.CHECK_OUT,
      amenities: [],
      images: [],
    },
  });

  const amenities = watch('amenities');
  const images = watch('images');

  useEffect(() => {
    if (!isAuthenticated || !isAdmin) {
      navigate(ROUTES.LOGIN);
      return;
    }

    if (isEditing) {
      fetchHotel();
    }
  }, [isAuthenticated, isAdmin, id, navigate]);

  const fetchHotel = async () => {
    try {
      setFetchLoading(true);
      const hotel = await hotelsService.getById(id);
      reset({
        name: hotel.name || '',
        description: hotel.description || '',
        address: hotel.address || '',
        city: hotel.city || '',
        state: hotel.state || '',
        country: hotel.country || '',
        phone: hotel.phone || '',
        email: hotel.email || '',
        price_per_night: hotel.price_per_night || hotel.pricePerNight || '',
        rating: hotel.rating || '',
        avaiable_rooms: hotel.avaiable_rooms || hotel.avaiableRooms || '',
        check_in_time: hotel.check_in_time || hotel.checkInTime || DEFAULT_TIMES.CHECK_IN,
        check_out_time: hotel.check_out_time || hotel.checkOutTime || DEFAULT_TIMES.CHECK_OUT,
        amenities: hotel.amenities || [],
        images: hotel.images || [],
      });
    } catch (err) {
      console.error('Error fetching hotel:', err);
      setError('Could not load hotel information');
    } finally {
      setFetchLoading(false);
    }
  };

  const onSubmit = async (data) => {
    try {
      setLoading(true);
      setError(null);

      const hotelData = {
        name: data.name,
        description: data.description,
        address: data.address,
        city: data.city,
        state: data.state,
        country: data.country,
        phone: data.phone,
        email: data.email,
        price_per_night: parseFloat(data.price_per_night) || 0,
        rating: parseFloat(data.rating) || 0,
        avaiable_rooms: parseInt(data.avaiable_rooms) || 0,
        check_in_time: data.check_in_time,
        check_out_time: data.check_out_time,
        amenities: data.amenities,
        images: data.images,
      };

      if (isEditing) {
        await adminService.updateHotel(id, hotelData);
        setSnackbar({ open: true, message: 'Hotel updated successfully', severity: 'success' });
      } else {
        await adminService.createHotel(hotelData);
        setSnackbar({ open: true, message: 'Hotel created successfully', severity: 'success' });
      }

      setTimeout(() => navigate(ROUTES.ADMIN), 1500);
    } catch (err) {
      console.error('Error saving hotel:', err);
      setError(err.response?.data?.error || 'Error saving hotel');
    } finally {
      setLoading(false);
    }
  };

  const handleAddAmenity = () => {
    if (amenityInput.trim() && !amenities.includes(amenityInput.trim())) {
      setValue('amenities', [...amenities, amenityInput.trim()]);
      setAmenityInput('');
    }
  };

  const handleRemoveAmenity = (amenity) => {
    setValue('amenities', amenities.filter((a) => a !== amenity));
  };

  const handleAddImage = () => {
    if (imageInput.trim() && !images.includes(imageInput.trim())) {
      setValue('images', [...images, imageInput.trim()]);
      setImageInput('');
    }
  };

  const handleRemoveImage = (image) => {
    setValue('images', images.filter((i) => i !== image));
  };

  if (!isAuthenticated || !isAdmin) {
    return null;
  }

  if (fetchLoading) {
    return (
      <Container maxWidth="md" sx={{ py: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <CircularProgress />
        </Box>
      </Container>
    );
  }

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Header */}
      <Box sx={{ bgcolor: 'primary.main', py: 4 }}>
        <Container maxWidth="md">
          <Button
            startIcon={<ArrowBackIcon />}
            onClick={() => navigate(ROUTES.ADMIN)}
            sx={{ color: 'white', mb: 2 }}
          >
            Back to Dashboard
          </Button>
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <HotelIcon sx={{ fontSize: 36, color: 'secondary.main', mr: 2 }} />
            <Typography variant="h4" sx={{ color: 'white', fontWeight: 600 }}>
              {isEditing ? 'Edit Hotel' : 'New Hotel'}
            </Typography>
          </Box>
        </Container>
      </Box>

      <Container maxWidth="md" sx={{ py: 4 }}>
        {error && (
          <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        <Card>
          <CardContent sx={{ p: 4 }}>
            <form onSubmit={handleSubmit(onSubmit)}>
              <Typography variant="h6" sx={{ mb: 3, fontWeight: 600 }}>
                Basic Information
              </Typography>

              <Grid container spacing={3}>
                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    label="Hotel Name"
                    {...register('name', { required: 'Name is required' })}
                    error={!!errors.name}
                    helperText={errors.name?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    label="Description"
                    multiline
                    rows={4}
                    {...register('description')}
                  />
                </Grid>

                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    label="Address"
                    {...register('address', { required: 'Address is required' })}
                    error={!!errors.address}
                    helperText={errors.address?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="City"
                    {...register('city', { required: 'City is required' })}
                    error={!!errors.city}
                    helperText={errors.city?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="State/Province"
                    {...register('state')}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Country"
                    {...register('country', { required: 'Country is required' })}
                    error={!!errors.country}
                    helperText={errors.country?.message}
                  />
                </Grid>
              </Grid>

              <Typography variant="h6" sx={{ mt: 4, mb: 3, fontWeight: 600 }}>
                Contact
              </Typography>

              <Grid container spacing={3}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Phone"
                    {...register('phone')}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Email"
                    type="email"
                    {...register('email')}
                  />
                </Grid>
              </Grid>

              <Typography variant="h6" sx={{ mt: 4, mb: 3, fontWeight: 600 }}>
                Pricing & Availability
              </Typography>

              <Grid container spacing={3}>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Price per Night"
                    type="number"
                    InputProps={{
                      startAdornment: <InputAdornment position="start">$</InputAdornment>,
                    }}
                    {...register('price_per_night', {
                      required: 'Price is required',
                      min: { value: 0, message: 'Price must be greater than 0' },
                    })}
                    error={!!errors.price_per_night}
                    helperText={errors.price_per_night?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Rating"
                    type="number"
                    inputProps={{ step: 0.1, min: VALIDATION.MIN_RATING, max: VALIDATION.MAX_RATING }}
                    {...register('rating', {
                      min: { value: VALIDATION.MIN_RATING, message: 'Minimum 0' },
                      max: { value: VALIDATION.MAX_RATING, message: 'Maximum 5' },
                    })}
                    error={!!errors.rating}
                    helperText={errors.rating?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Available Rooms"
                    type="number"
                    inputProps={{ min: 0 }}
                    {...register('avaiable_rooms', {
                      required: 'This field is required',
                      min: { value: 0, message: 'Minimum 0' },
                    })}
                    error={!!errors.avaiable_rooms}
                    helperText={errors.avaiable_rooms?.message}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Check-in Time"
                    type="time"
                    InputLabelProps={{ shrink: true }}
                    {...register('check_in_time')}
                  />
                </Grid>

                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Check-out Time"
                    type="time"
                    InputLabelProps={{ shrink: true }}
                    {...register('check_out_time')}
                  />
                </Grid>
              </Grid>

              <Typography variant="h6" sx={{ mt: 4, mb: 3, fontWeight: 600 }}>
                Amenities
              </Typography>

              <Box sx={{ mb: 2 }}>
                <TextField
                  size="small"
                  placeholder="Add amenity (e.g., WiFi, Pool, Gym)"
                  value={amenityInput}
                  onChange={(e) => setAmenityInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddAmenity())}
                  InputProps={{
                    endAdornment: (
                      <InputAdornment position="end">
                        <IconButton onClick={handleAddAmenity} size="small">
                          <AddIcon />
                        </IconButton>
                      </InputAdornment>
                    ),
                  }}
                  sx={{ width: 300 }}
                />
              </Box>

              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
                {amenities.map((amenity, index) => (
                  <Chip
                    key={index}
                    label={amenity}
                    onDelete={() => handleRemoveAmenity(amenity)}
                    color="primary"
                    variant="outlined"
                  />
                ))}
              </Box>

              <Typography variant="h6" sx={{ mt: 4, mb: 3, fontWeight: 600 }}>
                Images
              </Typography>

              <Box sx={{ mb: 2 }}>
                <TextField
                  size="small"
                  placeholder="Image URL"
                  value={imageInput}
                  onChange={(e) => setImageInput(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddImage())}
                  InputProps={{
                    endAdornment: (
                      <InputAdornment position="end">
                        <IconButton onClick={handleAddImage} size="small">
                          <AddIcon />
                        </IconButton>
                      </InputAdornment>
                    ),
                  }}
                  fullWidth
                />
              </Box>

              <Grid container spacing={2}>
                {images.map((image, index) => (
                  <Grid key={index} size={{ xs: 6, sm: 4, md: 3 }}>
                    <Box sx={{ position: 'relative' }}>
                      <Box
                        component="img"
                        src={image}
                        alt={`Image ${index + 1}`}
                        sx={{
                          width: '100%',
                          height: 100,
                          objectFit: 'cover',
                          borderRadius: 1,
                        }}
                      />
                      <IconButton
                        size="small"
                        onClick={() => handleRemoveImage(image)}
                        sx={{
                          position: 'absolute',
                          top: 4,
                          right: 4,
                          bgcolor: 'error.main',
                          color: 'white',
                          '&:hover': { bgcolor: 'error.dark' },
                        }}
                      >
                        <CloseIcon fontSize="small" />
                      </IconButton>
                    </Box>
                  </Grid>
                ))}
              </Grid>

              <Box sx={{ mt: 4, display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
                <Button
                  variant="outlined"
                  onClick={() => navigate(ROUTES.ADMIN)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="contained"
                  startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <SaveIcon />}
                  disabled={loading}
                >
                  {isEditing ? 'Save Changes' : 'Create Hotel'}
                </Button>
              </Box>
            </form>
          </CardContent>
        </Card>
      </Container>

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

export default HotelForm;
