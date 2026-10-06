/**
 * Skip link (plan 13 fase 8): primer elemento tabulable; visible solo al
 * foco, salta al <main id="main-content">.
 */

import { Box } from '@mui/material';

const SkipLink = () => (
  <Box
    component="a"
    href="#main-content"
    sx={{
      position: 'absolute',
      left: 12,
      top: -48,
      zIndex: (theme) => theme.zIndex.appBar + 1,
      bgcolor: 'primary.main',
      color: 'primary.contrastText',
      px: 2,
      py: 1,
      borderRadius: 1,
      textDecoration: 'none',
      transition: 'top 150ms ease',
      '&:focus-visible': { top: 12 },
    }}
  >
    Skip to main content
  </Box>
);

export default SkipLink;
