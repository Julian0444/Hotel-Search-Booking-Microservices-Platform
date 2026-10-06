/**
 * Marca propia (plan 13 fase 4): monograma SVG inline + wordmark serif.
 * Code-native, sin depender de icon fonts ni assets remotos.
 */

import { Box, Typography } from '@mui/material';
import { Link } from 'react-router';
import { ROUTES } from '../../constants';

export const BrandGlyph = ({ size = 34 }) => (
  <Box
    component="svg"
    viewBox="0 0 32 32"
    aria-hidden
    sx={{ width: size, height: size, display: 'block', flexShrink: 0 }}
  >
    <rect x="1.5" y="1.5" width="29" height="29" rx="6" fill="none" stroke="currentColor" strokeWidth="2" />
    <path
      d="M10 22V10h2.6l3.4 6.2L19.4 10H22v12h-2.4v-7.4L16.6 20h-1.2l-3-5.4V22H10z"
      fill="currentColor"
    />
  </Box>
);

// Sin aria-label: el nombre accesible es el texto visible del wordmark —
// un aria-label distinto viola label-in-name (WCAG 2.5.3)
const BrandMark = ({ to = ROUTES.HOME, compact = false }) => (
  <Box
    component={Link}
    to={to}
    sx={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 1.25,
      textDecoration: 'none',
      color: 'primary.main',
    }}
  >
    <Box sx={{ color: 'secondary.dark', display: 'flex' }}>
      <BrandGlyph size={compact ? 28 : 34} />
    </Box>
    <Box>
      <Typography
        variant={compact ? 'h6' : 'h5'}
        component="span"
        sx={{
          fontFamily: '"Cormorant Garamond", Georgia, serif',
          fontWeight: 700,
          lineHeight: 1,
          letterSpacing: '0.01em',
          display: 'block',
        }}
      >
        StayLux
      </Typography>
      {!compact && (
        <Typography
          variant="caption"
          component="span"
          sx={{ color: 'text.secondary', letterSpacing: '0.18em', fontSize: '0.6rem', display: 'block' }}
        >
          STAY · RESERVE · MANAGE
        </Typography>
      )}
    </Box>
  </Box>
);

export default BrandMark;
