/**
 * Register Page (plan 13 fase 2): siempre crea cuentas de cliente (los
 * admins se siembran server-side). Autocomplete de password nuevo, errores
 * con aria-live y foco al primer campo inválido.
 */

import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
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
} from '@mui/material';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useForm } from 'react-hook-form';
import { useAuth } from '../hooks/useAuth';
import { ROUTES, VALIDATION } from '../constants';
import BrandMark from '../components/common/BrandMark';
import RouteMeta from '../components/common/RouteMeta';

const Register = () => {
  const navigate = useNavigate();
  const { register: registerUser, isSubmitting } = useAuth();
  const [showPassword, setShowPassword] = useState(false);
  const [formError, setFormError] = useState(null);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
    setFocus,
  } = useForm();

  const password = watch('password');

  const onSubmit = async (data) => {
    setFormError(null);
    const result = await registerUser(data.username.trim(), data.password);
    if (result.success) {
      navigate(ROUTES.HOME);
    } else {
      setFormError(result.error);
      setFocus('username');
    }
  };

  const onInvalid = (fieldErrors) => {
    const first = Object.keys(fieldErrors)[0];
    if (first) setFocus(first);
  };

  return (
    <Box sx={{ minHeight: '100vh', display: 'flex', bgcolor: 'primary.main' }}>
      <RouteMeta title="Create account" description="Create a StayLux account to reserve stays." />
      <Container
        maxWidth="sm"
        sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', py: 4 }}
      >
        <Paper
          elevation={6}
          component="main"
          id="main-content"
          sx={{ width: '100%', p: { xs: 3, sm: 5 }, borderRadius: 3 }}
        >
          <Box sx={{ textAlign: 'center', mb: 4 }}>
            <BrandMark />
          </Box>

          <Typography variant="h4" component="h1" sx={{ textAlign: 'center', fontWeight: 600, mb: 1 }}>
            Create your account
          </Typography>
          <Typography variant="body1" color="text.secondary" sx={{ textAlign: 'center', mb: 4 }}>
            Search, reserve and manage your stays
          </Typography>

          <Box aria-live="polite">
            {formError && (
              <Alert severity="error" onClose={() => setFormError(null)} sx={{ mb: 3 }}>
                {formError}
              </Alert>
            )}
          </Box>

          <form onSubmit={handleSubmit(onSubmit, onInvalid)} noValidate>
            <TextField
              fullWidth
              label="Username"
              autoComplete="username"
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
              sx={{ mb: 3 }}
            />

            <TextField
              fullWidth
              type={showPassword ? 'text' : 'password'}
              label="Password"
              autoComplete="new-password"
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
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton
                      aria-label={showPassword ? 'Hide password' : 'Show password'}
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
              type="password"
              label="Confirm password"
              autoComplete="new-password"
              {...register('confirmPassword', {
                required: 'Please confirm your password',
                validate: (value) => value === password || 'Passwords do not match',
              })}
              error={!!errors.confirmPassword}
              helperText={errors.confirmPassword?.message}
              sx={{ mb: 4 }}
            />

            <Button
              type="submit"
              fullWidth
              variant="contained"
              size="large"
              disabled={isSubmitting}
              sx={{ py: 1.5, mb: 3 }}
            >
              {isSubmitting ? (
                <CircularProgress size={24} color="inherit" aria-label="Creating account" />
              ) : (
                'Create account'
              )}
            </Button>
          </form>

          <Divider sx={{ my: 3 }}>
            <Typography variant="caption" color="text.secondary">
              Already have an account?
            </Typography>
          </Divider>

          <Button component={Link} to={ROUTES.LOGIN} fullWidth variant="outlined" size="large">
            Sign in
          </Button>
        </Paper>
      </Container>
    </Box>
  );
};

export default Register;
