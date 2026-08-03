/**
 * Imagen con aspect ratio estable, lazy y fallback local (plan 13 fases 3/8):
 * una URL rota cae al placeholder conservando alt y proporción — cero layout
 * shift.
 */

import { useState, useEffect } from 'react';
import { Box } from '@mui/material';
import { HOTEL_FALLBACK_IMAGE } from '../../constants';

const ResponsiveImage = ({
  src,
  alt,
  aspectRatio = '4 / 3',
  fallback = HOTEL_FALLBACK_IMAGE,
  priority = false,
  sx = {},
}) => {
  const [failed, setFailed] = useState(false);

  // Si cambia la fuente (paginación, edición), reintentar con la nueva
  useEffect(() => setFailed(false), [src]);

  const effectiveSrc = failed || !src ? fallback : src;

  return (
    <Box sx={{ aspectRatio, overflow: 'hidden', bgcolor: 'action.hover', ...sx }}>
      <Box
        component="img"
        src={effectiveSrc}
        alt={alt}
        loading={priority ? 'eager' : 'lazy'}
        decoding="async"
        onError={() => setFailed(true)}
        sx={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block' }}
      />
    </Box>
  );
};

export default ResponsiveImage;
