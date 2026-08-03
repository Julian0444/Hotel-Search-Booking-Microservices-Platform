/**
 * Estado vacío estándar (plan 13 fase 3): distinto de un error — dice qué
 * pasó y ofrece la próxima acción.
 */

import { Box, Button, Typography } from '@mui/material';

const EmptyState = ({ icon = null, title, description, actionLabel, onAction }) => (
  <Box sx={{ textAlign: 'center', py: { xs: 6, md: 10 }, px: 3 }}>
    {icon && (
      <Box sx={{ color: 'divider', mb: 2, '& svg': { fontSize: 72 } }} aria-hidden>
        {icon}
      </Box>
    )}
    <Typography variant="h5" component="h2" gutterBottom>
      {title}
    </Typography>
    {description && (
      <Typography variant="body1" color="text.secondary" sx={{ mb: actionLabel ? 3 : 0 }}>
        {description}
      </Typography>
    )}
    {actionLabel && onAction && (
      <Button variant="contained" onClick={onAction}>
        {actionLabel}
      </Button>
    )}
  </Box>
);

export default EmptyState;
