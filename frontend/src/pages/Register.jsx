/**
 * Register Page
 * New user registration form
 */

import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Paper,
  Typography,
  TextField,
  Button,
  Alert,
  InputAdornment,
  IconButton,
  Divider,
  CircularProgress,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import {
  Visibility,
  VisibilityOff,
  Person as PersonIcon,
  Lock as LockIcon,
  Hotel as HotelIcon,
  Badge as BadgeIcon,
} from '@mui/icons-material';
import { useForm, Controller } from 'react-hook-form';
import { useAuth } from '../context/AuthContext';
import { ROUTES, VALIDATION, USER_ROLES } from '../constants';

const Register = () => {
  const navigate = useNavigate();
  const { register: registerUser, loading, error, setError } = useAuth();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const {
    register,
    handleSubmit,
    control,
    watch,
    formState: { errors },
  } = useForm({
    defaultValues: {
      tipo: USER_ROLES.CLIENT,
    },
  });

  const password = watch('password');

  const onSubmit = async (data) => {
    const result = await registerUser(data.username, data.password, data.tipo);
    if (result.success) {
      navigate(ROUTES.HOME);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        position: 'relative',
        '&::before': {
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundImage: 'url(https://images.unsplash.com/photo-1551882547-ff40c63fe5fa?w=1920)',
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          filter: 'brightness(0.35)',
          zIndex: 0,
        },
      }}
    >
      <Container
        maxWidth="sm"
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          py: 4,
          position: 'relative',
          zIndex: 1,
        }}
      >
        <Paper
          elevation={24}
          sx={{
            width: '100%',
            p: { xs: 3, sm: 5 },
            borderRadius: 3,
            bgcolor: 'rgba(255,255,255,0.97)',
          }}
        >
          {/* Logo */}
          <Box sx={{ textAlign: 'center', mb: 4 }}>
            <Box
              component={Link}
              to={ROUTES.HOME}
              sx={{
                display: 'inline-flex',
                alignItems: 'center',
                textDecoration: 'none',
                color: 'inherit',
              }}
            >
              <HotelIcon sx={{ fontSize: 40, color: 'secondary.main', mr: 1 }} />
              <Typography variant="h4" sx={{ fontWeight: 700, color: 'primary.main' }}>
                StayLux
              </Typography>
            </Box>
          </Box>

          <Typography variant="h4" sx={{ textAlign: 'center', fontWeight: 600, mb: 1 }}>
            Create Account
          </Typography>
          <Typography
            variant="body1"
            color="text.secondary"
            sx={{ textAlign: 'center', mb: 4 }}
          >
            Join StayLux and discover unique experiences
          </Typography>

          {error && (
            <Alert
              severity="error"
              onClose={() => setError(null)}
              sx={{ mb: 3 }}
            >
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit(onSubmit)}>
            <TextField
              fullWidth
              label="Username"
              placeholder="Choose a username"
              {...register('username', {
                required: 'Username is required',
                minLength: {
                  value: VALIDATION.MIN_USERNAME_LENGTH,
                  message: `Username must be at least ${VALIDATION.MIN_USERNAME_LENGTH} characters`,
                },
                pattern: {
                  value: /^[a-zA-Z0-9_]+$/,
                  message: 'Username can only contain letters, numbers, and underscores',
                },
              })}
              error={!!errors.username}
              helperText={errors.username?.message}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <PersonIcon color="action" />
                  </InputAdornment>
                ),
              }}
              sx={{ mb: 3 }}
            />

            <TextField
              fullWidth
              type={showPassword ? 'text' : 'password'}
              label="Password"
              placeholder="Create a secure password"
              {...register('password', {
                required: 'Password is required',
                minLength: {
                  value: VALIDATION.MIN_PASSWORD_LENGTH,
                  message: `Password must be at least ${VALIDATION.MIN_PASSWORD_LENGTH} characters`,
                },
              })}
              error={!!errors.password}
              helperText={errors.password?.message}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <LockIcon color="action" />
                  </InputAdornment>
                ),
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      onClick={() => setShowPassword(!showPassword)}
                      edge="end"
                    >
                      {showPassword ? <VisibilityOff /> : <Visibility />}
                    </IconButton>
                  </InputAdornment>
                ),
              }}
              sx={{ mb: 3 }}
            />

            <TextField
              fullWidth
              type={showConfirmPassword ? 'text' : 'password'}
              label="Confirm Password"
              placeholder="Repeat your password"
              {...register('confirmPassword', {
                required: 'Please confirm your password',
                validate: (value) =>
                  value === password || 'Passwords do not match',
              })}
              error={!!errors.confirmPassword}
              helperText={errors.confirmPassword?.message}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <LockIcon color="action" />
                  </InputAdornment>
                ),
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                      edge="end"
                    >
                      {showConfirmPassword ? <VisibilityOff /> : <Visibility />}
                    </IconButton>
                  </InputAdornment>
                ),
              }}
              sx={{ mb: 3 }}
            />

            <Controller
              name="tipo"
              control={control}
              render={({ field }) => (
                <FormControl fullWidth sx={{ mb: 4 }}>
                  <InputLabel>Account Type</InputLabel>
                  <Select
                    {...field}
                    label="Account Type"
                    startAdornment={
                      <InputAdornment position="start">
                        <BadgeIcon color="action" />
                      </InputAdornment>
                    }
                  >
                    <MenuItem value={USER_ROLES.CLIENT}>Customer - Search and book hotels</MenuItem>
                    <MenuItem value={USER_ROLES.ADMIN}>Administrator - Manage hotels</MenuItem>
                  </Select>
                </FormControl>
              )}
            />

            <Button
              type="submit"
              fullWidth
              variant="contained"
              size="large"
              disabled={loading}
              sx={{ py: 1.5, mb: 3 }}
            >
              {loading ? (
                <CircularProgress size={24} color="inherit" />
              ) : (
                'Create Account'
              )}
            </Button>
          </form>

          <Divider sx={{ my: 3 }}>
            <Typography variant="caption" color="text.secondary">
              Already have an account?
            </Typography>
          </Divider>

          <Button
            component={Link}
            to={ROUTES.LOGIN}
            fullWidth
            variant="outlined"
            size="large"
            sx={{ borderWidth: 2 }}
          >
            Sign In
          </Button>
        </Paper>
      </Container>
    </Box>
  );
};

export default Register;
