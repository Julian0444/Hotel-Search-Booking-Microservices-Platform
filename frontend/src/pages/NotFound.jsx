/**
 * 404 real (plan 13 fase 8): útil — buscador y regreso — en vez del
 * redirect silencioso a Home que había antes.
 */

import { useNavigate, Link } from 'react-router';
import { Box, Button, Container, Typography } from '@mui/material';
import { SearchBar } from '../components/Hotels';
import RouteMeta from '../components/common/RouteMeta';
import { ROUTES } from '../constants';

const NotFound = () => {
  const navigate = useNavigate();

  return (
    <Container maxWidth="sm" sx={{ py: { xs: 8, md: 12 }, textAlign: 'center' }}>
      <RouteMeta title="Page not found" description="The page you were looking for does not exist." />
      <Typography variant="overline" color="text.secondary">
        404
      </Typography>
      <Typography variant="h3" component="h1" sx={{ mb: 2 }}>
        That page does not exist
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        The address may be mistyped, or the page may have moved. Try searching for a stay instead.
      </Typography>

      <Box sx={{ mb: 4 }}>
        <SearchBar
          compact
          onSearch={(q) => navigate(q ? `${ROUTES.SEARCH}?q=${encodeURIComponent(q)}` : ROUTES.SEARCH)}
        />
      </Box>

      <Box sx={{ display: 'flex', gap: 2, justifyContent: 'center' }}>
        <Button component={Link} to={ROUTES.HOME} variant="contained">
          Back to home
        </Button>
        <Button component={Link} to={ROUTES.SEARCH} variant="outlined">
          Browse all stays
        </Button>
      </Box>
    </Container>
  );
};

export default NotFound;
