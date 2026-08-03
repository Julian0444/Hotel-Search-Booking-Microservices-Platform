/**
 * Search Page (plan 13 fase 3, RV22/F13-03).
 * La URL (q/page/sort) es la única fuente de verdad; el count y las páginas
 * salen de meta.total (índice completo); el sort lo aplica Solr globalmente.
 * keepPreviousData mantiene el grid al paginar; skeleton solo en la primera
 * carga; empty, error y retry son estados distintos.
 */

import { useEffect } from 'react';
import {
  Box,
  Chip,
  Container,
  Fade,
  FormControl,
  Grid,
  InputLabel,
  LinearProgress,
  MenuItem,
  Pagination,
  Select,
  Skeleton,
  Typography,
} from '@mui/material';
import { HotelOutlined as HotelIcon } from '@mui/icons-material';
import { SearchBar, HotelCard, HotelGridSkeleton } from '../components/Hotels';
import { EmptyState, ErrorState, RouteMeta } from '../components/common';
import { useHotelSearch } from '../hooks/queries';
import { useHotelSearchParams } from '../hooks/useHotelSearchParams';
import { SORT_OPTIONS } from '../constants';

const Search = () => {
  const { q, page, sort, limit, offset, setQuery, setPage, setSort } = useHotelSearchParams();

  const { data, isPending, isFetching, isError, error, refetch } = useHotelSearch({
    q,
    offset,
    limit,
    sort,
  });

  const total = data?.total ?? 0;
  const hotels = data?.items ?? [];
  const totalPages = Math.max(1, Math.ceil(total / limit));

  // Si el total bajó (hotel borrado, query cambiada por URL editada) y la
  // página quedó fuera de rango, corregir la URL en vez de mostrar vacío
  useEffect(() => {
    if (data && page > totalPages) {
      setPage(totalPages);
    }
  }, [data, page, totalPages, setPage]);

  const handlePageChange = (_event, value) => {
    setPage(value);
    window.scrollTo({ top: 0 });
  };

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      <RouteMeta
        title={q ? `“${q}” stays` : 'Search stays'}
        description="Search the indexed hotel catalog by city, country or name."
      />

      {/* Header */}
      <Box sx={{ bgcolor: 'primary.main', py: { xs: 4, md: 6 } }}>
        <Container maxWidth="lg">
          <Typography variant="h3" component="h1" sx={{ color: 'white', fontWeight: 600, mb: 0.5 }}>
            Search stays
          </Typography>
          <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.75)', mb: 3 }}>
            Full-text search over the indexed catalog
          </Typography>
          <SearchBar onSearch={setQuery} initialQuery={q} />
        </Container>
      </Box>

      {/* Results */}
      <Container maxWidth="lg" component="section" sx={{ py: 4 }}>
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            mb: 3,
            flexWrap: 'wrap',
            gap: 2,
          }}
        >
          <Typography variant="body1" color="text.secondary" aria-live="polite">
            {isPending ? (
              <Skeleton width={140} />
            ) : (
              <>
                <strong>{total}</strong> stay{total === 1 ? '' : 's'}
                {q && (
                  <>
                    {' '}for “{q}”
                    <Chip label={q} size="small" onDelete={() => setQuery('')} sx={{ ml: 1 }} />
                  </>
                )}
              </>
            )}
          </Typography>

          <FormControl size="small" sx={{ minWidth: 190 }}>
            <InputLabel id="sort-label">Sort by</InputLabel>
            <Select
              labelId="sort-label"
              value={sort}
              label="Sort by"
              onChange={(event) => setSort(event.target.value)}
            >
              {SORT_OPTIONS.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Box>

        {/* Indicador sutil al re-buscar con el grid anterior visible */}
        <Fade in={isFetching && !isPending} unmountOnExit>
          <LinearProgress aria-label="Updating results" sx={{ mb: 2, borderRadius: 1 }} />
        </Fade>

        {isError ? (
          <ErrorState error={error} title="Search is unavailable" onRetry={refetch} />
        ) : isPending ? (
          <HotelGridSkeleton count={6} />
        ) : hotels.length === 0 ? (
          <EmptyState
            icon={<HotelIcon />}
            title="No stays match that search"
            description="Try a different city, country, or hotel name."
            actionLabel={q ? 'Clear search' : undefined}
            onAction={q ? () => setQuery('') : undefined}
          />
        ) : (
          <>
            <Grid container spacing={3}>
              {hotels.map((hotel) => (
                <Grid key={hotel.id} size={{ xs: 12, sm: 6, md: 4 }}>
                  <HotelCard hotel={hotel} headingComponent="h2" />
                </Grid>
              ))}
            </Grid>

            {totalPages > 1 && (
              <Box component="nav" aria-label="Search results pages" sx={{ display: 'flex', justifyContent: 'center', mt: 6 }}>
                <Pagination
                  count={totalPages}
                  page={Math.min(page, totalPages)}
                  onChange={handlePageChange}
                  color="primary"
                  size="large"
                  showFirstButton
                  showLastButton
                />
              </Box>
            )}
          </>
        )}
      </Container>
    </Box>
  );
};

export default Search;
