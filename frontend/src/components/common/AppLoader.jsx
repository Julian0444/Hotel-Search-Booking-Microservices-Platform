/**
 * Loader accesible de bootstrap/lazy (plan 13 fases 2/9): reserva la
 * altura del viewport para no saltar layout y se anuncia a lectores.
 */

import { Box, CircularProgress, Typography } from '@mui/material';

const AppLoader = ({ label = 'Loading' }) => (
  <Box
    role="status"
    aria-live="polite"
    sx={{
      minHeight: '60vh',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      gap: 2,
    }}
  >
    <CircularProgress size={32} aria-hidden />
    <Typography variant="body2" color="text.secondary">
      {label}…
    </Typography>
  </Box>
);

export default AppLoader;
