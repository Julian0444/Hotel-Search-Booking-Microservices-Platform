/**
 * Hotel Card Component
 * Displays hotel information in a card format
 */

import { Link } from 'react-router-dom';
import {
  Card,
  CardMedia,
  CardContent,
  Typography,
  Box,
  Chip,
  Rating,
  Button,
} from '@mui/material';
import {
  LocationOn as LocationIcon,
  Wifi as WifiIcon,
  Pool as PoolIcon,
  Restaurant as RestaurantIcon,
  FitnessCenter as GymIcon,
  Spa as SpaIcon,
  LocalParking as ParkingIcon,
} from '@mui/icons-material';
import { getHotelImage, formatPrice } from '../../utils/helpers';

const amenityIcons = {
  wifi: <WifiIcon sx={{ fontSize: 16 }} />,
  pool: <PoolIcon sx={{ fontSize: 16 }} />,
  restaurant: <RestaurantIcon sx={{ fontSize: 16 }} />,
  gym: <GymIcon sx={{ fontSize: 16 }} />,
  spa: <SpaIcon sx={{ fontSize: 16 }} />,
  parking: <ParkingIcon sx={{ fontSize: 16 }} />,
};

const HotelCard = ({ hotel }) => {
  const imageUrl = getHotelImage(hotel.id, hotel.images);
  const pricePerNight = hotel.price_per_night || hotel.pricePerNight || 0;
  const availableRooms = hotel.avaiable_rooms || hotel.avaiableRooms || 0;

  return (
    <Card
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        '&:hover': { transform: 'translateY(-4px)' },
        transition: 'all 0.3s ease',
      }}
    >
      <Box sx={{ position: 'relative' }}>
        <CardMedia
          component="img"
          height="220"
          image={imageUrl}
          alt={hotel.name}
          sx={{ objectFit: 'cover' }}
        />
        <Box
          sx={{
            position: 'absolute',
            top: 12,
            right: 12,
            bgcolor: 'secondary.main',
            color: 'primary.main',
            px: 2,
            py: 0.5,
            borderRadius: 1,
            fontWeight: 700,
          }}
        >
          <Typography variant="subtitle2" fontWeight={700}>
            {formatPrice(pricePerNight)}/night
          </Typography>
        </Box>
      </Box>

      <CardContent sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', p: 3 }}>
        <Typography
          variant="h6"
          component="h3"
          sx={{
            fontWeight: 600,
            mb: 1,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {hotel.name}
        </Typography>

        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1.5, color: 'text.secondary' }}>
          <LocationIcon sx={{ fontSize: 18, mr: 0.5 }} />
          <Typography variant="body2" noWrap>
            {hotel.city}, {hotel.country}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <Rating value={hotel.rating || 0} precision={0.5} size="small" readOnly sx={{ mr: 1 }} />
          <Typography variant="body2" color="text.secondary">
            ({hotel.rating?.toFixed(1) || '0.0'})
          </Typography>
        </Box>

        <Typography
          variant="body2"
          color="text.secondary"
          sx={{
            mb: 2,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            lineHeight: 1.6,
          }}
        >
          {hotel.description || 'Discover this amazing hotel with all the amenities you need for a perfect stay.'}
        </Typography>

        {hotel.amenities && hotel.amenities.length > 0 && (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 2 }}>
            {hotel.amenities.slice(0, 4).map((amenity, index) => (
              <Chip
                key={index}
                size="small"
                icon={amenityIcons[amenity.toLowerCase()] || null}
                label={amenity}
                sx={{ bgcolor: 'background.default', fontSize: '0.7rem', height: 24 }}
              />
            ))}
            {hotel.amenities.length > 4 && (
              <Chip
                size="small"
                label={`+${hotel.amenities.length - 4}`}
                sx={{ bgcolor: 'primary.light', color: 'white', fontSize: '0.7rem', height: 24 }}
              />
            )}
          </Box>
        )}

        <Box sx={{ mt: 'auto', pt: 2, borderTop: '1px solid', borderColor: 'divider' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="caption" color="text.secondary">
              {availableRooms} rooms available
            </Typography>
            <Button
              component={Link}
              to={`/hotels/${hotel.id}`}
              variant="contained"
              size="small"
              sx={{ minWidth: 100 }}
            >
              View Details
            </Button>
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
};

export default HotelCard;
