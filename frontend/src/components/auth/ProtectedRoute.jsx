/**
 * ProtectedRoute (plan 13 fase 2, F13-02).
 * - Bootstrap: loader accesible, no un frame en blanco.
 * - Anónimo: a /login con state.from — el login vuelve a la intención.
 * - Cliente en ruta admin: pantalla 403 explícita, nunca un frame del
 *   dashboard ni un redirect mudo.
 */

import { Navigate, useLocation, Link } from 'react-router';
import { Box, Button, Container, Typography } from '@mui/material';
import { useAuth } from '../../hooks/useAuth';
import { ROUTES } from '../../constants';
import AppLoader from '../common/AppLoader';

const Forbidden = () => (
  <Container maxWidth="sm" sx={{ py: 10, textAlign: 'center' }}>
    <Typography variant="overline" color="text.secondary">
      403
    </Typography>
    <Typography variant="h4" component="h1" sx={{ mb: 2 }}>
      This area is for administrators
    </Typography>
    <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
      Your account does not have permission to view the admin panel.
    </Typography>
    <Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
      <Button component={Link} to={ROUTES.HOME} variant="contained">
        Back to home
      </Button>
      <Button component={Link} to={ROUTES.SEARCH} variant="outlined">
        Browse stays
      </Button>
    </Box>
  </Container>
);

const ProtectedRoute = ({ children, adminOnly = false }) => {
  const { isAuthenticated, isAdmin, isBootstrapping } = useAuth();
  const location = useLocation();

  if (isBootstrapping) {
    return <AppLoader label="Checking your session" />;
  }

  if (!isAuthenticated) {
    return <Navigate to={ROUTES.LOGIN} replace state={{ from: location }} />;
  }

  if (adminOnly && !isAdmin) {
    return <Forbidden />;
  }

  return children;
};

export default ProtectedRoute;
