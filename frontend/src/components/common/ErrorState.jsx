/**
 * Estado de error estándar (plan 13): mensaje estable del ApiError, retry
 * propio y — solo si existe — la referencia técnica (trace_id) copiable en
 * una línea secundaria. Nunca internals del response.
 */

import { Box, Button, Typography } from '@mui/material';
import { ErrorOutline as ErrorIcon } from '@mui/icons-material';

const ErrorState = ({ error, title = 'Something went wrong', onRetry, compact = false }) => {
  const message = error?.message || 'Please try again in a moment.';
  const traceId = error?.traceId;

  return (
    <Box
      role="alert"
      sx={{ textAlign: 'center', py: compact ? 3 : { xs: 6, md: 10 }, px: 3 }}
    >
      <ErrorIcon sx={{ fontSize: compact ? 40 : 64, color: 'warning.main', mb: 1 }} aria-hidden />
      <Typography variant={compact ? 'h6' : 'h5'} component="h2" gutterBottom>
        {title}
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 2 }}>
        {message}
      </Typography>
      {traceId && (
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ display: 'block', mb: 2, fontFamily: 'monospace', userSelect: 'all' }}
        >
          Reference: {traceId}
        </Typography>
      )}
      {onRetry && (
        <Button variant="outlined" onClick={onRetry}>
          Try again
        </Button>
      )}
    </Box>
  );
};

export default ErrorState;
