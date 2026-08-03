/**
 * Footer (plan 13 fase 4): compacto y honesto — navegación real, link al
 * repositorio y nada más. Sin links muertos, sin social falso, sin datos de
 * contacto inventados.
 */

import { Box, Container, Divider, Link as MuiLink, Stack, Typography } from '@mui/material';
import { Link } from 'react-router';
import { GitHub as GitHubIcon } from '@mui/icons-material';
import { ROUTES, REPO_URL } from '../../constants';
import { BrandGlyph } from '../common/BrandMark';

const footerLinkSx = {
  color: 'rgba(255,255,255,0.85)',
  textDecoration: 'none',
  '&:hover': { color: 'secondary.main' },
};

const Footer = () => (
  <Box component="footer" sx={{ bgcolor: 'primary.main', color: 'white', mt: 'auto', py: { xs: 4, md: 5 } }}>
    <Container maxWidth="lg">
      <Stack
        direction={{ xs: 'column', md: 'row' }}
        spacing={{ xs: 3, md: 6 }}
        justifyContent="space-between"
        alignItems={{ xs: 'flex-start', md: 'center' }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Box sx={{ color: 'secondary.main', display: 'flex' }}>
            <BrandGlyph size={30} />
          </Box>
          <Box>
            <Typography variant="h6" component="p" sx={{ fontFamily: '"Cormorant Garamond", Georgia, serif', lineHeight: 1.1 }}>
              StayLux
            </Typography>
            <Typography variant="caption" sx={{ opacity: 0.7 }}>
              Indexed search · protected availability · idempotent booking
            </Typography>
          </Box>
        </Box>

        <Stack component="nav" aria-label="Footer" direction="row" spacing={3} flexWrap="wrap">
          <MuiLink component={Link} to={ROUTES.HOME} sx={footerLinkSx}>
            Home
          </MuiLink>
          <MuiLink component={Link} to={ROUTES.SEARCH} sx={footerLinkSx}>
            Search
          </MuiLink>
          <MuiLink component={Link} to={ROUTES.RESERVATIONS} sx={footerLinkSx}>
            My reservations
          </MuiLink>
          <MuiLink
            href={REPO_URL}
            target="_blank"
            rel="noreferrer"
            sx={{ ...footerLinkSx, display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
          >
            <GitHubIcon sx={{ fontSize: 18 }} aria-hidden />
            Source code
          </MuiLink>
        </Stack>
      </Stack>

      <Divider sx={{ borderColor: 'rgba(255,255,255,0.12)', my: 3 }} />

      <Typography variant="body2" sx={{ opacity: 0.65 }}>
        Portfolio project — three Go microservices behind an nginx gateway, with event-driven search.
      </Typography>
    </Container>
  </Box>
);

export default Footer;
