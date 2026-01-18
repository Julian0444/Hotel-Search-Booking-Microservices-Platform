/**
 * Login Page
 * User authentication form
 */

import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
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
import {
  Visibility,
  VisibilityOff,
  Person as PersonIcon,
  Lock as LockIcon,
  Hotel as HotelIcon,
} from '@mui/icons-material';
import { useForm } from 'react-hook-form';
import { useAuth } from '../context/AuthContext';
import { ROUTES, VALIDATION } from '../constants';

const Login = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { login, loading, error, setError } = useAuth();
  const [showPassword, setShowPassword] = useState(false);

  const from = location.state?.from?.pathname || ROUTES.HOME;

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();

  const onSubmit = async (data) => {
    const result = await login(data.username, data.password);
    if (result.success) {
      navigate(from, { replace: true });
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
          backgroundImage: 'url(https://images.unsplash.com/photo-1520250497591-112f2f40a3f4?w=1920)',
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
            Welcome Back
          </Typography>
          <Typography
            variant="body1"
            color="text.secondary"
            sx={{ textAlign: 'center', mb: 4 }}
          >
            Sign in to access your account
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
              placeholder="Enter your username"
              {...register('username', {
                required: 'Username is required',
                minLength: {
                  value: VALIDATION.MIN_USERNAME_LENGTH,
                  message: `Username must be at least ${VALIDATION.MIN_USERNAME_LENGTH} characters`,
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
              placeholder="Enter your password"
              {...register('password', {
                required: 'Password is required',
                minLength: {
                  value: 4,
                  message: 'Password must be at least 4 characters',
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
              sx={{ mb: 4 }}
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
                'Sign In'
              )}
            </Button>
          </form>

          <Divider sx={{ my: 3 }}>
            <Typography variant="caption" color="text.secondary">
              Don't have an account?
            </Typography>
          </Divider>

          <Button
            component={Link}
            to={ROUTES.REGISTER}
            fullWidth
            variant="outlined"
            size="large"
            sx={{ borderWidth: 2 }}
          >
            Create Account
          </Button>
        </Paper>
      </Container>
    </Box>
  );
};

export default Login;
