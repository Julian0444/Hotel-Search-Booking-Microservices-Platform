/**
 * Search Page
 * Hotel search results with filtering and pagination
 */

import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Grid,
  Card,
  CardContent,
  Skeleton,
  Alert,
  Pagination,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
} from '@mui/material';
import { Hotel as HotelIcon } from '@mui/icons-material';
import { SearchBar, HotelCard } from '../components/Hotels';
import { hotelsService } from '../services';
import { PAGINATION, SORT_OPTIONS } from '../constants';

const Search = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get('q') || '';

  const [hotels, setHotels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [sortBy, setSortBy] = useState(SORT_OPTIONS.RELEVANCE);
  const [query, setQuery] = useState(initialQuery);

  useEffect(() => {
    const fetchHotels = async () => {
      try {
        setLoading(true);
        setError(null);
        const offset = (page - 1) * PAGINATION.DEFAULT_PAGE_SIZE;
        const response = await hotelsService.search(query, offset, PAGINATION.DEFAULT_PAGE_SIZE);

        let hotelList = response || [];

        // Sort based on selected criteria
        if (sortBy === SORT_OPTIONS.PRICE_LOW) {
          hotelList.sort((a, b) => (a.price_per_night || a.pricePerNight || 0) - (b.price_per_night || b.pricePerNight || 0));
        } else if (sortBy === SORT_OPTIONS.PRICE_HIGH) {
          hotelList.sort((a, b) => (b.price_per_night || b.pricePerNight || 0) - (a.price_per_night || a.pricePerNight || 0));
        } else if (sortBy === SORT_OPTIONS.RATING) {
          hotelList.sort((a, b) => (b.rating || 0) - (a.rating || 0));
        }

        setHotels(hotelList);
        setTotalPages(Math.max(1, Math.ceil(hotelList.length / PAGINATION.DEFAULT_PAGE_SIZE)));
      } catch (err) {
        console.error('Error searching hotels:', err);
        setError('Could not load hotels. Please try again later.');
        setHotels([]);
      } finally {
        setLoading(false);
      }
    };

    fetchHotels();
  }, [query, page, sortBy]);

  const handleSearch = (newQuery) => {
    setQuery(newQuery);
    setPage(1);
    setSearchParams(newQuery ? { q: newQuery } : {});
  };

  const handlePageChange = (event, value) => {
    setPage(value);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <Box sx={{ bgcolor: 'background.default', minHeight: '100vh' }}>
      {/* Header */}
      <Box
        sx={{
          bgcolor: 'primary.main',
          py: 6,
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            opacity: 0.1,
            backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='1'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E")`,
          }}
        />
        <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 4 }}>
            <HotelIcon sx={{ fontSize: 40, color: 'secondary.main', mr: 2 }} />
            <Box>
              <Typography variant="h3" sx={{ color: 'white', fontWeight: 600 }}>
                Search Hotels
              </Typography>
              <Typography variant="body1" sx={{ color: 'rgba(255,255,255,0.7)' }}>
                Find your perfect accommodation
              </Typography>
            </Box>
          </Box>
          <SearchBar onSearch={handleSearch} initialQuery={query} />
        </Container>
      </Box>

      {/* Results */}
      <Container maxWidth="lg" sx={{ py: 4 }}>
        {/* Filters and count */}
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            mb: 4,
            flexWrap: 'wrap',
            gap: 2,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Typography variant="body1" color="text.secondary">
              {loading ? (
                <Skeleton width={150} />
              ) : (
                <>
                  <strong>{hotels.length}</strong> hotels found
                  {query && (
                    <>
                      {' '}for "{query}"
                      <Chip
                        label={query}
                        size="small"
                        onDelete={() => handleSearch('')}
                        sx={{ ml: 1 }}
                      />
                    </>
                  )}
                </>
              )}
            </Typography>
          </Box>

          <FormControl size="small" sx={{ minWidth: 180 }}>
            <InputLabel>Sort by</InputLabel>
            <Select
              value={sortBy}
              label="Sort by"
              onChange={(e) => setSortBy(e.target.value)}
            >
              <MenuItem value={SORT_OPTIONS.RELEVANCE}>Relevance</MenuItem>
              <MenuItem value={SORT_OPTIONS.PRICE_LOW}>Price: Low to High</MenuItem>
              <MenuItem value={SORT_OPTIONS.PRICE_HIGH}>Price: High to Low</MenuItem>
              <MenuItem value={SORT_OPTIONS.RATING}>Top Rated</MenuItem>
            </Select>
          </FormControl>
        </Box>

        {/* Error message */}
        {error && (
          <Alert severity="error" sx={{ mb: 4 }}>
            {error}
          </Alert>
        )}

        {/* Hotel grid */}
        <Grid container spacing={3}>
          {loading
            ? Array.from({ length: 8 }).map((_, index) => (
                <Grid key={index} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                  <Card>
                    <Skeleton variant="rectangular" height={200} />
                    <CardContent>
                      <Skeleton variant="text" height={28} width="80%" />
                      <Skeleton variant="text" height={20} width="60%" />
                      <Skeleton variant="text" height={18} width="40%" />
                      <Skeleton variant="text" height={50} />
                    </CardContent>
                  </Card>
                </Grid>
              ))
            : hotels.map((hotel) => (
                <Grid key={hotel.id} size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
                  <HotelCard hotel={hotel} />
                </Grid>
              ))}
        </Grid>

        {/* No results */}
        {!loading && hotels.length === 0 && !error && (
          <Box
            sx={{
              textAlign: 'center',
              py: 8,
              px: 4,
            }}
          >
            <HotelIcon sx={{ fontSize: 80, color: 'divider', mb: 2 }} />
            <Typography variant="h5" gutterBottom>
              No hotels found
            </Typography>
            <Typography variant="body1" color="text.secondary">
              Try different search terms or explore our available options.
            </Typography>
          </Box>
        )}

        {/* Pagination */}
        {!loading && hotels.length > 0 && totalPages > 1 && (
          <Box sx={{ display: 'flex', justifyContent: 'center', mt: 6 }}>
            <Pagination
              count={totalPages}
              page={page}
              onChange={handlePageChange}
              color="primary"
              size="large"
              showFirstButton
              showLastButton
            />
          </Box>
        )}
      </Container>
    </Box>
  );
};

export default Search;
