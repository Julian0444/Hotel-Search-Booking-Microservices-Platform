/**
 * Home (plan 13 fase 4): tres actos — hero editorial con búsqueda,
 * selección real del catálogo, "how it works" verificable — más una franja
 * discreta de arquitectura. Cero claims inventados: todo lo visible se
 * puede demostrar en la app.
 */

import { useNavigate, Link } from 'react-router';
import { alpha } from '@mui/material/styles';
import { Box, Button, Container, Grid, Typography } from '@mui/material';
import {
  SearchOutlined as SearchIcon,
  EventAvailableOutlined as ReserveIcon,
  ManageHistoryOutlined as ManageIcon,
  ArrowForward as ArrowIcon,
} from '@mui/icons-material';
import { SearchBar, HotelCard, HotelGridSkeleton } from '../components/Hotels';
import { ErrorState, RouteMeta } from '../components/common';
import { useHotelSearch } from '../hooks/queries';
import { ROUTES, REPO_URL } from '../constants';
import { tokens } from '../theme/theme';

const steps = [
  {
    icon: <SearchIcon sx={{ fontSize: 32 }} />,
    title: 'Search',
    description: 'Full-text search over a Solr index that stays in sync with the catalog through events.',
  },
  {
    icon: <ReserveIcon sx={{ fontSize: 32 }} />,
    title: 'Reserve',
    description: 'Per-night availability is claimed atomically — no overbooking, and retries never double-book.',
  },
  {
    icon: <ManageIcon sx={{ fontSize: 32 }} />,
    title: 'Manage',
    description: 'Your reservation history keeps every booking, including cancelled ones, with real statuses.',
  },
];

const Home = () => {
  const navigate = useNavigate();
  const { data, isPending, isError, error, refetch } = useHotelSearch({ q: '', offset: 0, limit: 6 });
  const featured = data?.items ?? [];

  const handleSearch = (query) => {
    navigate(query ? `${ROUTES.SEARCH}?q=${encodeURIComponent(query)}` : ROUTES.SEARCH);
  };

  return (
    <Box>
      <RouteMeta
        title="Stays worth returning to"
        description="Search indexed stays, reserve with protected availability, and manage your bookings."
      />

      {/* Acto 1 — hero editorial */}
      <Box
        component="section"
        aria-labelledby="hero-heading"
        sx={{
          bgcolor: 'primary.main',
          color: 'white',
          position: 'relative',
          overflow: 'hidden',
          py: { xs: 8, md: 12 },
          // Trama editorial sutil, inline — sin imágenes remotas
          '&::before': {
            content: '""',
            position: 'absolute',
            inset: 0,
            opacity: 0.35,
            background: `radial-gradient(1200px 500px at 85% -10%, ${alpha(tokens.gold, 0.35)}, transparent 60%),
                         radial-gradient(800px 400px at -10% 110%, ${alpha(tokens.inkLight, 0.9)}, transparent 55%)`,
          },
        }}
      >
        <Container maxWidth="lg" sx={{ position: 'relative' }}>
          <Box sx={{ maxWidth: 760 }}>
            <Typography
              variant="overline"
              component="p"
              sx={{ color: 'secondary.main', letterSpacing: '0.2em', mb: 2 }}
            >
              STAYLUX — SEARCH · RESERVE · MANAGE
            </Typography>
            <Typography id="hero-heading" variant="h1" component="h1" sx={{ mb: 3, lineHeight: 1.08 }}>
              Stays worth returning to
            </Typography>
            <Typography variant="h6" component="p" sx={{ color: 'rgba(255,255,255,0.82)', fontWeight: 400, mb: 5, maxWidth: 560 }}>
              Browse a curated catalog — from Villa Carlos Paz to El Calafate — with instant search,
              protected availability and a booking flow you can trust twice.
            </Typography>
            <Box sx={{ maxWidth: 640 }}>
              <SearchBar onSearch={handleSearch} />
            </Box>
          </Box>
        </Container>
      </Box>

      {/* Acto 2 — selección real del catálogo */}
      <Box component="section" aria-labelledby="featured-heading" sx={{ py: { xs: 6, md: 9 } }}>
        <Container maxWidth="lg">
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'flex-end',
              mb: 4,
              flexWrap: 'wrap',
              gap: 2,
            }}
          >
            <Box>
              <Typography variant="overline" component="p" sx={{ color: 'secondary.dark', letterSpacing: '0.15em' }}>
                FROM THE CATALOG
              </Typography>
              <Typography id="featured-heading" variant="h3" component="h2" sx={{ mt: 0.5 }}>
                Featured stays
              </Typography>
            </Box>
            <Button
              component={Link}
              to={ROUTES.SEARCH}
              variant="outlined"
              endIcon={<ArrowIcon />}
            >
              Browse all
            </Button>
          </Box>

          {isError ? (
            <ErrorState compact error={error} title="The catalog is unavailable" onRetry={refetch} />
          ) : isPending ? (
            <HotelGridSkeleton count={6} />
          ) : featured.length === 0 ? (
            <Typography color="text.secondary">
              No stays in the catalog yet — an administrator can add the first one.
            </Typography>
          ) : (
            <Grid container spacing={3}>
              {featured.map((hotel) => (
                <Grid key={hotel.id} size={{ xs: 12, sm: 6, md: 4 }}>
                  <HotelCard hotel={hotel} />
                </Grid>
              ))}
            </Grid>
          )}
        </Container>
      </Box>

      {/* Acto 3 — how it works, verificable */}
      <Box component="section" aria-labelledby="how-heading" sx={{ bgcolor: 'background.paper', py: { xs: 6, md: 9 } }}>
        <Container maxWidth="lg">
          <Typography variant="overline" component="p" sx={{ color: 'secondary.dark', letterSpacing: '0.15em', textAlign: 'center' }}>
            HOW IT WORKS
          </Typography>
          <Typography id="how-heading" variant="h3" component="h2" sx={{ textAlign: 'center', mt: 0.5, mb: 6 }}>
            Three steps, no surprises
          </Typography>

          <Grid container spacing={4}>
            {steps.map((step, index) => (
              <Grid key={step.title} size={{ xs: 12, md: 4 }}>
                <Box sx={{ textAlign: 'center', px: 2 }}>
                  <Box
                    sx={{
                      width: 64,
                      height: 64,
                      borderRadius: '50%',
                      border: '1.5px solid',
                      borderColor: 'secondary.main',
                      color: 'secondary.dark',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      mx: 'auto',
                      mb: 2,
                    }}
                    aria-hidden
                  >
                    {step.icon}
                  </Box>
                  <Typography variant="h6" component="h3" sx={{ mb: 1 }}>
                    {index + 1}. {step.title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {step.description}
                  </Typography>
                </Box>
              </Grid>
            ))}
          </Grid>
        </Container>
      </Box>

      {/* Franja de arquitectura — la única "promesa externa" y es el repo */}
      <Box component="section" sx={{ py: { xs: 4, md: 5 }, borderTop: '1px solid', borderColor: 'divider' }}>
        <Container maxWidth="lg" sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}>
          <Typography variant="body2" color="text.secondary">
            Built as three Go microservices behind an nginx gateway — MySQL, MongoDB, Solr, RabbitMQ and
            layered caching, wired with event-driven sync.
          </Typography>
          <Button href={REPO_URL} target="_blank" rel="noreferrer" size="small" variant="text">
            Read the architecture →
          </Button>
        </Container>
      </Box>
    </Box>
  );
};

export default Home;
