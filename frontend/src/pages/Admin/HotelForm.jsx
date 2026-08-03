/**
 * Hotel Form (plan 13 fase 7, F13-08): contrato estricto del plan 07/11 —
 * "HH:mm", price numérico, available_rooms entero, URLs http(s) —, amenities
 * de catálogo + opción libre controlada, summary de errores con focus al
 * primero, cambios sucios protegidos y navegación SIN setTimeout.
 */

import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router';
import {
  Alert,
  AlertTitle,
  Autocomplete,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  InputAdornment,
  TextField,
  Typography,
} from '@mui/material';
import { Save as SaveIcon, ArrowBack as ArrowBackIcon } from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { useHotel } from '../../hooks/queries';
import { useSaveHotel } from '../../hooks/mutations';
import { useUnsavedChanges } from '../../hooks/useUnsavedChanges';
import HotelImageFields from '../../components/admin/HotelImageFields';
import { AppLoader, ErrorState, RouteMeta } from '../../components/common';
import { isTimeOfDay, isOptionalEmail } from '../../utils/validators';
import { amenityLabel } from '../../utils/helpers';
import { ROUTES, DEFAULT_TIMES, VALIDATION, AMENITY_CATALOG } from '../../constants';

const emptyValues = {
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
  available_rooms: '',
  check_in_time: DEFAULT_TIMES.CHECK_IN,
  check_out_time: DEFAULT_TIMES.CHECK_OUT,
  amenities: [],
  images: [],
};

const hotelToValues = (hotel) => ({
  ...emptyValues,
  ...Object.fromEntries(
    Object.entries({
      name: hotel.name,
      description: hotel.description,
      address: hotel.address,
      city: hotel.city,
      state: hotel.state,
      country: hotel.country,
      phone: hotel.phone,
      email: hotel.email,
      price_per_night: hotel.price_per_night,
      rating: hotel.rating,
      available_rooms: hotel.available_rooms,
      check_in_time: hotel.check_in_time || DEFAULT_TIMES.CHECK_IN,
      check_out_time: hotel.check_out_time || DEFAULT_TIMES.CHECK_OUT,
      amenities: hotel.amenities ?? [],
      images: hotel.images ?? [],
    }).map(([key, value]) => [key, value ?? '']),
  ),
});

const HotelForm = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const isEditing = !!id;
  const [saved, setSaved] = useState(null); // {id} tras crear/editar

  const hotelQuery = useHotel(id, { enabled: isEditing });
  const saveMutation = useSaveHotel();

  const {
    register,
    control,
    handleSubmit,
    reset,
    setFocus,
    formState: { errors, isDirty },
  } = useForm({ defaultValues: emptyValues, mode: 'onTouched' });

  const unsaved = useUnsavedChanges(isDirty && !saved);

  // Editar: hidratar el form cuando llega el hotel
  useEffect(() => {
    if (isEditing && hotelQuery.data) {
      reset(hotelToValues(hotelQuery.data));
    }
  }, [isEditing, hotelQuery.data, reset]);

  const onSubmit = async (data) => {
    const payload = {
      name: data.name.trim(),
      description: data.description.trim(),
      address: data.address.trim(),
      city: data.city.trim(),
      state: data.state.trim(),
      country: data.country.trim(),
      phone: data.phone.trim(),
      email: data.email.trim(),
      price_per_night: Number(data.price_per_night),
      rating: data.rating === '' ? 0 : Number(data.rating),
      available_rooms: Number(data.available_rooms),
      check_in_time: data.check_in_time,
      check_out_time: data.check_out_time,
      amenities: data.amenities,
      images: data.images,
    };

    try {
      const result = await saveMutation.mutateAsync({ hotelId: id, data: payload });
      reset(data); // limpia isDirty: no pedir confirmación al salir
      setSaved({ id: isEditing ? id : result.id });
    } catch {
      // saveMutation.error se muestra en el summary
    }
  };

  const onInvalid = (fieldErrors) => {
    const first = Object.keys(fieldErrors)[0];
    if (first) setFocus(first);
  };

  if (isEditing && hotelQuery.isPending) {
    return <AppLoader label="Loading hotel" />;
  }

  if (isEditing && hotelQuery.isError) {
    return (
      <Container maxWidth="md" sx={{ py: 6 }}>
        <ErrorState error={hotelQuery.error} title="We could not load this hotel" onRetry={hotelQuery.refetch} />
      </Container>
    );
  }

  const errorEntries = Object.entries(errors);

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      <RouteMeta
        title={isEditing ? 'Edit hotel' : 'New hotel'}
        description="Create or edit a hotel in the StayLux catalog."
      />

      <Box sx={{ bgcolor: 'primary.main', py: 4 }}>
        <Container maxWidth="md">
          <Button
            startIcon={<ArrowBackIcon />}
            onClick={() => unsaved.confirmLeave(() => navigate(ROUTES.ADMIN))}
            sx={{ color: 'white', mb: 1 }}
          >
            Back to dashboard
          </Button>
          <Typography variant="h4" component="h1" sx={{ color: 'white', fontWeight: 600 }}>
            {isEditing ? 'Edit hotel' : 'New hotel'}
          </Typography>
        </Container>
      </Box>

      <Container maxWidth="md" sx={{ py: 4 }}>
        <Box aria-live="polite">
          {errorEntries.length > 0 && (
            <Alert severity="error" sx={{ mb: 3 }}>
              <AlertTitle>Fix {errorEntries.length} field{errorEntries.length === 1 ? '' : 's'} before saving</AlertTitle>
              {errorEntries.map(([field, fieldError]) => (
                <Typography key={field} variant="body2">
                  • {fieldError.message}
                </Typography>
              ))}
            </Alert>
          )}
          {saveMutation.error && (
            <Alert severity="error" onClose={() => saveMutation.reset()} sx={{ mb: 3 }}>
              {saveMutation.error.message}
              {saveMutation.error.traceId && (
                <Typography variant="caption" sx={{ display: 'block', mt: 0.5, fontFamily: 'monospace' }}>
                  Reference: {saveMutation.error.traceId}
                </Typography>
              )}
            </Alert>
          )}
        </Box>

        <Card>
          <CardContent sx={{ p: { xs: 2.5, md: 4 } }}>
            <form onSubmit={handleSubmit(onSubmit, onInvalid)} noValidate>
              <Typography variant="h6" component="h2" sx={{ mb: 2.5 }}>
                Basics
              </Typography>
              <Grid container spacing={2.5}>
                <Grid size={{ xs: 12 }}>
                  <TextField
                    fullWidth
                    label="Hotel name"
                    {...register('name', { required: 'Hotel name is required' })}
                    error={!!errors.name}
                    helperText={errors.name?.message}
                  />
                </Grid>
                <Grid size={{ xs: 12 }}>
                  <TextField fullWidth label="Description" multiline rows={4} {...register('description')} />
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
                  <TextField fullWidth label="State / province" {...register('state')} />
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

              <Typography variant="h6" component="h2" sx={{ mt: 4, mb: 2.5 }}>
                Contact
              </Typography>
              <Grid container spacing={2.5}>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField fullWidth label="Phone" {...register('phone')} />
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Email"
                    type="email"
                    {...register('email', {
                      validate: (value) => isOptionalEmail(value) || 'Enter a valid email address',
                    })}
                    error={!!errors.email}
                    helperText={errors.email?.message}
                  />
                </Grid>
              </Grid>

              <Typography variant="h6" component="h2" sx={{ mt: 4, mb: 2.5 }}>
                Pricing and availability
              </Typography>
              <Grid container spacing={2.5}>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Price per night"
                    type="number"
                    inputProps={{ min: 0, step: '0.01' }}
                    InputProps={{ startAdornment: <InputAdornment position="start">$</InputAdornment> }}
                    {...register('price_per_night', {
                      required: 'Price per night is required',
                      validate: (value) =>
                        (Number(value) > 0 && Number.isFinite(Number(value))) || 'Price must be greater than 0',
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
                      validate: (value) =>
                        value === '' ||
                        (Number(value) >= VALIDATION.MIN_RATING && Number(value) <= VALIDATION.MAX_RATING) ||
                        `Rating must be between ${VALIDATION.MIN_RATING} and ${VALIDATION.MAX_RATING}`,
                    })}
                    error={!!errors.rating}
                    helperText={errors.rating?.message}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 4 }}>
                  <TextField
                    fullWidth
                    label="Available rooms"
                    type="number"
                    inputProps={{ min: 0, step: 1 }}
                    {...register('available_rooms', {
                      required: 'Available rooms is required',
                      validate: (value) =>
                        (Number.isInteger(Number(value)) && Number(value) >= 0) ||
                        'Available rooms must be a whole number of 0 or more',
                    })}
                    error={!!errors.available_rooms}
                    helperText={errors.available_rooms?.message}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Check-in time"
                    type="time"
                    InputLabelProps={{ shrink: true }}
                    {...register('check_in_time', {
                      validate: (value) => isTimeOfDay(value) || 'Check-in time must be HH:mm',
                    })}
                    error={!!errors.check_in_time}
                    helperText={errors.check_in_time?.message || '24h format, e.g. 15:00'}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Check-out time"
                    type="time"
                    InputLabelProps={{ shrink: true }}
                    {...register('check_out_time', {
                      validate: (value) => isTimeOfDay(value) || 'Check-out time must be HH:mm',
                    })}
                    error={!!errors.check_out_time}
                    helperText={errors.check_out_time?.message || '24h format, e.g. 11:00'}
                  />
                </Grid>
              </Grid>

              <Typography variant="h6" component="h2" sx={{ mt: 4, mb: 2.5 }}>
                Amenities
              </Typography>
              <Controller
                name="amenities"
                control={control}
                render={({ field }) => (
                  <Autocomplete
                    multiple
                    freeSolo
                    options={AMENITY_CATALOG}
                    getOptionLabel={amenityLabel}
                    value={field.value}
                    onChange={(_event, value) =>
                      field.onChange(
                        value.map((v) => v.trim().toLowerCase().replace(/\s+/g, '_')).filter(Boolean),
                      )
                    }
                    renderTags={(value, getTagProps) =>
                      value.map((option, index) => (
                        <Chip label={amenityLabel(option)} {...getTagProps({ index })} key={option} />
                      ))
                    }
                    renderInput={(params) => (
                      <TextField
                        {...params}
                        label="Amenities"
                        helperText="Pick from the catalog or type a custom one and press Enter"
                      />
                    )}
                  />
                )}
              />

              <Typography variant="h6" component="h2" sx={{ mt: 4, mb: 2.5 }}>
                Images
              </Typography>
              <Controller
                name="images"
                control={control}
                render={({ field }) => <HotelImageFields images={field.value} onChange={field.onChange} />}
              />

              <Box sx={{ mt: 4, display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
                <Button variant="outlined" onClick={() => unsaved.confirmLeave(() => navigate(ROUTES.ADMIN))}>
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="contained"
                  startIcon={saveMutation.isPending ? <CircularProgress size={18} color="inherit" /> : <SaveIcon />}
                  disabled={saveMutation.isPending}
                >
                  {isEditing ? 'Save changes' : 'Create hotel'}
                </Button>
              </Box>
            </form>
          </CardContent>
        </Card>
      </Container>

      {/* Éxito: acciones explícitas, sin temporizadores */}
      <Dialog open={!!saved} maxWidth="sm" fullWidth aria-labelledby="hotel-saved-title">
        <DialogTitle id="hotel-saved-title">
          {isEditing ? 'Changes saved' : 'Hotel created'}
        </DialogTitle>
        <DialogContent>
          <Typography variant="body1">
            {isEditing
              ? 'The hotel was updated and the search index will sync shortly.'
              : 'The hotel is now in the catalog and will appear in search shortly.'}
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button component={Link} to={`/hotels/${saved?.id}`} variant="outlined">
            View hotel
          </Button>
          <Button onClick={() => navigate(ROUTES.ADMIN)} variant="contained">
            Back to dashboard
          </Button>
        </DialogActions>
      </Dialog>

      {/* Cambios sin guardar */}
      <Dialog open={unsaved.isConfirming} onClose={unsaved.stay} maxWidth="xs" fullWidth aria-labelledby="unsaved-title">
        <DialogTitle id="unsaved-title">Discard unsaved changes?</DialogTitle>
        <DialogContent>
          <Typography variant="body2" color="text.secondary">
            Your edits to this hotel have not been saved.
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={unsaved.stay} variant="outlined">
            Keep editing
          </Button>
          <Button onClick={unsaved.discardAndLeave} color="error" variant="contained">
            Discard changes
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default HotelForm;
