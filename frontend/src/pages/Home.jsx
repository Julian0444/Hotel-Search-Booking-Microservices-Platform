/**
 * Home Page
 * Landing page with hero section, features, and featured hotels
 */

import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Typography,
  Grid,
  Button,
  Card,
  CardContent,
  Skeleton,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Verified as VerifiedIcon,
  Support as SupportIcon,
  Security as SecurityIcon,
} from '@mui/icons-material';
import { SearchBar, HotelCard } from '../components/Hotels';
import { hotelsService } from '../services';
import { ROUTES } from '../constants';

const features = [
  {
    icon: <SearchIcon sx={{ fontSize: 40 }} />,
    title: 'Smart Search',
    description: 'Find the perfect hotel with our advanced search system by city, country, or hotel name.',
  },
  {
    icon: <VerifiedIcon sx={{ fontSize: 40 }} />,
    title: 'Verified Hotels',
    description: 'All our hotels undergo a rigorous verification process to guarantee quality.',
  },
  {
    icon: <SupportIcon sx={{ fontSize: 40 }} />,
    title: '24/7 Support',
    description: 'Our customer service team is available around the clock to assist you.',
  },
  {
    icon: <SecurityIcon sx={{ fontSize: 40 }} />,
    title: 'Secure Booking',
    description: 'Your information is protected with the highest security standards.',
  },
];

const Home = () => {
  const navigate = useNavigate();
  const [featuredHotels, setFeaturedHotels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchFeaturedHotels = async () => {
      try {
        setLoading(true);
        const response = await hotelsService.search('', 0, 6);
        setFeaturedHotels(response || []);
      } catch (err) {
        console.error('Error fetching hotels:', err);
        setError('Could not load featured hotels');
      } finally {
        setLoading(false);
      }
    };

    fetchFeaturedHotels();
  }, []);

  const handleSearch = (query) => {
    navigate(`${ROUTES.SEARCH}?q=${encodeURIComponent(query)}`);
  };

  return (
    <Box>
      {/* Hero Section */}
      <Box
        sx={{
          position: 'relative',
          minHeight: { xs: '70vh', md: '80vh' },
          display: 'flex',
          alignItems: 'center',
          overflow: 'hidden',
          '&::before': {
            content: '""',
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundImage: 'url(https://images.unsplash.com/photo-1564501049412-61c2a3083791?w=1920)',
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: 'brightness(0.4)',
            zIndex: 0,
          },
        }}
      >
        <Container maxWidth="lg" sx={{ position: 'relative', zIndex: 1 }}>
          <Box sx={{ maxWidth: 800, mx: 'auto', textAlign: 'center' }}>
            <Typography
              variant="overline"
              sx={{
                color: 'secondary.main',
                letterSpacing: '0.2em',
                mb: 2,
                display: 'block',
              }}
            >
              WELCOME TO STAYLUX
            </Typography>
            <Typography
              variant="h1"
              sx={{
                color: 'white',
                fontSize: { xs: '2.5rem', md: '4rem', lg: '5rem' },
                fontWeight: 700,
                mb: 3,
                lineHeight: 1.1,
              }}
            >
              Discover Your Next Unforgettable Experience
            </Typography>
            <Typography
              variant="h6"
              sx={{
                color: 'rgba(255,255,255,0.8)',
                mb: 5,
                fontWeight: 400,
                maxWidth: 600,
                mx: 'auto',
              }}
            >
              Explore the world's finest hotels and book with confidence, 
              knowing every detail has been designed for you.
            </Typography>

            <Box sx={{ maxWidth: 700, mx: 'auto' }}>
              <SearchBar onSearch={handleSearch} />
            </Box>
          </Box>
        </Container>

        {/* Decorative gradient overlay */}
        <Box
          sx={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            right: 0,
            height: 200,
            background: 'linear-gradient(to top, rgba(247,245,242,1) 0%, rgba(247,245,242,0) 100%)',
            zIndex: 1,
          }}
        />
      </Box>

      {/* Features Section */}
      <Container maxWidth="lg" sx={{ py: 10 }}>
        <Box sx={{ textAlign: 'center', mb: 8 }}>
          <Typography
            variant="overline"
            sx={{ color: 'secondary.main', letterSpacing: '0.15em' }}
          >
            WHY CHOOSE US
          </Typography>
          <Typography variant="h3" sx={{ mt: 1, fontWeight: 600 }}>
            Excellence in Every Detail
          </Typography>
        </Box>

        <Grid container spacing={4}>
          {features.map((feature, index) => (
            <Grid key={index} size={{ xs: 12, sm: 6, md: 3 }}>
              <Card
                elevation={0}
                sx={{
                  height: '100%',
                  textAlign: 'center',
                  bgcolor: 'transparent',
                  transition: 'all 0.3s ease',
                  '&:hover': {
                    bgcolor: 'white',
                    boxShadow: '0 8px 40px rgba(26, 54, 93, 0.1)',
                  },
                }}
              >
                <CardContent sx={{ p: 4 }}>
                  <Box
                    sx={{
                      width: 80,
                      height: 80,
                      borderRadius: '50%',
                      bgcolor: 'primary.main',
                      color: 'white',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      mx: 'auto',
                      mb: 3,
                    }}
                  >
                    {feature.icon}
                  </Box>
                  <Typography variant="h6" sx={{ mb: 2, fontWeight: 600 }}>
                    {feature.title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {feature.description}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      </Container>

      {/* Featured Hotels Section */}
      <Box sx={{ bgcolor: 'white', py: 10 }}>
        <Container maxWidth="lg">
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'flex-end',
              mb: 6,
              flexWrap: 'wrap',
              gap: 2,
            }}
          >
            <Box>
              <Typography
                variant="overline"
                sx={{ color: 'secondary.main', letterSpacing: '0.15em' }}
              >
                FEATURED DESTINATIONS
              </Typography>
              <Typography variant="h3" sx={{ mt: 1, fontWeight: 600 }}>
                Popular Hotels
              </Typography>
            </Box>
            <Button
              variant="outlined"
              size="large"
              onClick={() => navigate(ROUTES.SEARCH)}
              sx={{ borderWidth: 2 }}
            >
              View All
            </Button>
          </Box>

          {error && (
            <Alert severity="info" sx={{ mb: 4 }}>
              {error}. Showing sample data.
            </Alert>
          )}

          <Grid container spacing={4}>
            {loading
              ? Array.from({ length: 6 }).map((_, index) => (
                  <Grid key={index} size={{ xs: 12, sm: 6, md: 4 }}>
                    <Card>
                      <Skeleton variant="rectangular" height={220} />
                      <CardContent>
                        <Skeleton variant="text" height={32} width="80%" />
                        <Skeleton variant="text" height={24} width="60%" />
                        <Skeleton variant="text" height={20} width="40%" />
                        <Skeleton variant="text" height={60} />
                      </CardContent>
                    </Card>
                  </Grid>
                ))
              : featuredHotels.map((hotel) => (
                  <Grid key={hotel.id} size={{ xs: 12, sm: 6, md: 4 }}>
                    <HotelCard hotel={hotel} />
                  </Grid>
                ))}
          </Grid>
        </Container>
      </Box>

      {/* CTA Section */}
      <Box
        sx={{
          position: 'relative',
          py: 12,
          overflow: 'hidden',
          '&::before': {
            content: '""',
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundImage: 'url(https://images.unsplash.com/photo-1571003123894-1f0594d2b5d9?w=1920)',
            backgroundSize: 'cover',
            backgroundPosition: 'center',
            filter: 'brightness(0.3)',
            zIndex: 0,
          },
        }}
      >
        <Container maxWidth="md" sx={{ position: 'relative', zIndex: 1, textAlign: 'center' }}>
          <Typography
            variant="h2"
            sx={{ color: 'white', fontWeight: 600, mb: 3 }}
          >
            Ready for Your Next Adventure?
          </Typography>
          <Typography
            variant="h6"
            sx={{ color: 'rgba(255,255,255,0.8)', mb: 4, fontWeight: 400 }}
          >
            Join thousands of travelers who trust StayLux to find 
            the best hospitality experiences.
          </Typography>
          <Button
            variant="contained"
            size="large"
            onClick={() => navigate(ROUTES.REGISTER)}
            sx={{
              bgcolor: 'secondary.main',
              color: 'primary.main',
              px: 5,
              py: 1.5,
              '&:hover': {
                bgcolor: 'secondary.dark',
              },
            }}
          >
            Create Free Account
          </Button>
        </Container>
      </Box>
    </Box>
  );
};

export default Home;
