/**
 * RouteAnnouncer (plan 13 fase 8): al cambiar de ruta mueve el foco al main
 * y anuncia el título por live region — sin robar el foco mientras el
 * usuario tipea (solo actúa si el pathname cambió de verdad).
 */

import { useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router';
import { Box } from '@mui/material';

const RouteAnnouncer = () => {
  const location = useLocation();
  const [announcement, setAnnouncement] = useState('');
  const previousPath = useRef(location.pathname);

  useEffect(() => {
    if (previousPath.current === location.pathname) return;
    previousPath.current = location.pathname;

    // El <title> por ruta lo pone RouteMeta; anunciar tras el paint
    const id = window.setTimeout(() => {
      setAnnouncement(document.title);
      const main = document.getElementById('main-content');
      if (main) {
        main.focus({ preventScroll: false });
      }
      window.scrollTo({ top: 0, behavior: 'auto' });
    }, 80);

    return () => window.clearTimeout(id);
  }, [location.pathname]);

  return (
    <Box
      aria-live="polite"
      aria-atomic="true"
      sx={{
        position: 'absolute',
        width: 1,
        height: 1,
        overflow: 'hidden',
        clip: 'rect(0 0 0 0)',
        whiteSpace: 'nowrap',
      }}
    >
      {announcement}
    </Box>
  );
};

export default RouteAnnouncer;
