/**
 * Footer Component
 * Site footer with navigation links and contact information
 */

import { Box, Container, Grid, Typography, IconButton, Link as MuiLink, Divider } from '@mui/material';
import { Link } from 'react-router-dom';
import {
  Hotel as HotelIcon,
  Email as EmailIcon,
  Phone as PhoneIcon,
  LocationOn as LocationIcon,
  Facebook as FacebookIcon,
  Instagram as InstagramIcon,
  Twitter as TwitterIcon,
} from '@mui/icons-material';
import { ROUTES } from '../../constants';

const Footer = () => {
  const currentYear = new Date().getFullYear();

  return (
    <Box
      component="footer"
      sx={{
        bgcolor: 'primary.main',
        color: 'white',
        mt: 'auto',
        pt: 8,
        pb: 4,
      }}
    >
      <Container maxWidth="lg">
        <Grid container spacing={6}>
          <Grid size={{ xs: 12, md: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
              <HotelIcon sx={{ fontSize: 36, color: 'secondary.main', mr: 1 }} />
              <Typography variant="h4" fontWeight={700}>
                StayLux
              </Typography>
            </Box>
            <Typography variant="body2" sx={{ opacity: 0.8, mb: 3, lineHeight: 1.8 }}>
              Discover unique experiences at the world's finest hotels. 
              Book with confidence and enjoy unforgettable stays with 
              our premium hospitality service.
            </Typography>
            <Box sx={{ display: 'flex', gap: 1 }}>
              <IconButton
                href="#"
                sx={{
                  color: 'white',
                  bgcolor: 'rgba(255,255,255,0.1)',
                  '&:hover': { bgcolor: 'secondary.main', color: 'primary.main' },
                }}
              >
                <FacebookIcon />
              </IconButton>
              <IconButton
                href="#"
                sx={{
                  color: 'white',
                  bgcolor: 'rgba(255,255,255,0.1)',
                  '&:hover': { bgcolor: 'secondary.main', color: 'primary.main' },
                }}
              >
                <InstagramIcon />
              </IconButton>
              <IconButton
                href="#"
                sx={{
                  color: 'white',
                  bgcolor: 'rgba(255,255,255,0.1)',
                  '&:hover': { bgcolor: 'secondary.main', color: 'primary.main' },
                }}
              >
                <TwitterIcon />
              </IconButton>
            </Box>
          </Grid>

          <Grid size={{ xs: 12, sm: 6, md: 2 }}>
            <Typography variant="overline" sx={{ color: 'secondary.main', mb: 2, display: 'block' }}>
              Navigation
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              <MuiLink
                component={Link}
                to={ROUTES.HOME}
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                Home
              </MuiLink>
              <MuiLink
                component={Link}
                to={ROUTES.SEARCH}
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                Search Hotels
              </MuiLink>
              <MuiLink
                component={Link}
                to={ROUTES.RESERVATIONS}
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                My Reservations
              </MuiLink>
            </Box>
          </Grid>

          <Grid size={{ xs: 12, sm: 6, md: 2 }}>
            <Typography variant="overline" sx={{ color: 'secondary.main', mb: 2, display: 'block' }}>
              Support
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              <MuiLink
                href="#"
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                Help Center
              </MuiLink>
              <MuiLink
                href="#"
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                Cancellation Policy
              </MuiLink>
              <MuiLink
                href="#"
                sx={{
                  color: 'white',
                  opacity: 0.8,
                  textDecoration: 'none',
                  '&:hover': { opacity: 1, color: 'secondary.main' },
                  transition: 'all 0.2s',
                }}
              >
                Terms & Conditions
              </MuiLink>
            </Box>
          </Grid>

          <Grid size={{ xs: 12, md: 4 }}>
            <Typography variant="overline" sx={{ color: 'secondary.main', mb: 2, display: 'block' }}>
              Contact Us
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ bgcolor: 'rgba(255,255,255,0.1)', p: 1, borderRadius: 1, display: 'flex' }}>
                  <LocationIcon sx={{ color: 'secondary.main' }} />
                </Box>
                <Typography variant="body2" sx={{ opacity: 0.8 }}>
                  123 Main Street, New York, NY
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ bgcolor: 'rgba(255,255,255,0.1)', p: 1, borderRadius: 1, display: 'flex' }}>
                  <PhoneIcon sx={{ color: 'secondary.main' }} />
                </Box>
                <Typography variant="body2" sx={{ opacity: 0.8 }}>
                  +1 (555) 123-4567
                </Typography>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                <Box sx={{ bgcolor: 'rgba(255,255,255,0.1)', p: 1, borderRadius: 1, display: 'flex' }}>
                  <EmailIcon sx={{ color: 'secondary.main' }} />
                </Box>
                <Typography variant="body2" sx={{ opacity: 0.8 }}>
                  contact@staylux.com
                </Typography>
              </Box>
            </Box>
          </Grid>
        </Grid>

        <Divider sx={{ borderColor: 'rgba(255,255,255,0.1)', my: 4 }} />

        <Box
          sx={{
            display: 'flex',
            flexDirection: { xs: 'column', sm: 'row' },
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: 2,
          }}
        >
          <Typography variant="body2" sx={{ opacity: 0.6 }}>
            © {currentYear} StayLux. All rights reserved.
          </Typography>
          <Typography variant="body2" sx={{ opacity: 0.6 }}>
            Made with ❤️ for unforgettable experiences
          </Typography>
        </Box>
      </Container>
    </Box>
  );
};

export default Footer;
