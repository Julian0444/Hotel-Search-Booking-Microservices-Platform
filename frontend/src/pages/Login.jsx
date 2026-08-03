/**
 * Login Page (plan 13 fase 2): explica una sesión expirada, vuelve a la
 * intención original (state.from) y separa el error del formulario del
 * estado global. Errores anunciados con aria-live y foco al primer campo.
 */

import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router';
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

const Login = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { login, isSubmitting, sessionNotice, clearSessionNotice, isAuthenticated } = useAuth();
  const [showPassword, setShowPassword] = useState(false);
  const [formError, setFormError] = useState(null);
  const usernameRef = useRef(null);

  const from = location.state?.from
    ? `${location.state.from.pathname || ''}${location.state.from.search || ''}` || ROUTES.HOME
    : ROUTES.HOME;

  const {
    register,
    handleSubmit,
    formState: { errors },
    setFocus,
  } = useForm();

  // Sesión ya activa (p. ej. Back hasta /login): a la home, no otro login
  useEffect(() => {
    if (isAuthenticated && !isSubmitting) {
      navigate(from, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const onSubmit = async (data) => {
    setFormError(null);
    const result = await login(data.username.trim(), data.password);
    if (result.success) {
      navigate(from, { replace: true });
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
      <RouteMeta title="Sign in" description="Sign in to manage your StayLux reservations." />
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
            Welcome back
          </Typography>
          <Typography variant="body1" color="text.secondary" sx={{ textAlign: 'center', mb: 4 }}>
            Sign in to manage your reservations
          </Typography>

          <Box aria-live="polite">
            {sessionNotice && (
              <Alert severity="info" onClose={clearSessionNotice} sx={{ mb: 3 }}>
                {sessionNotice}
              </Alert>
            )}
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
              inputRef={usernameRef}
              {...register('username', {
                required: 'Username is required',
                minLength: {
                  value: VALIDATION.MIN_USERNAME_LENGTH,
                  message: `Username must be at least ${VALIDATION.MIN_USERNAME_LENGTH} characters`,
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
              autoComplete="current-password"
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
              {isSubmitting ? <CircularProgress size={24} color="inherit" aria-label="Signing in" /> : 'Sign in'}
            </Button>
          </form>

          <Divider sx={{ my: 3 }}>
            <Typography variant="caption" color="text.secondary">
              New to StayLux?
            </Typography>
          </Divider>

          <Button component={Link} to={ROUTES.REGISTER} fullWidth variant="outlined" size="large">
            Create an account
          </Button>

          <Button component={Link} to={ROUTES.HOME} fullWidth sx={{ mt: 2 }}>
            Back to home
          </Button>
        </Paper>
      </Container>
    </Box>
  );
};

export default Login;
