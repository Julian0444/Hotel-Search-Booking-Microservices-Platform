/**
 * Hotel Card (plan 13 fases 3/4): contrato final sin fallbacks legacy,
 * imagen con aspect ratio estable + fallback local, rating con equivalente
 * textual y UNA sola acción de link (nada de links anidados).
 */

import { Link } from 'react-router';
import { Box, Card, CardContent, Chip, Rating, Typography } from '@mui/material';
import { LocationOn as LocationIcon } from '@mui/icons-material';
import { hotelImage, amenityLabel, truncateText } from '../../utils/helpers';
import { formatMajorAmount } from '../../utils/money';
import { prefetchHotelDetailChunk } from '../../utils/prefetch';
import ResponsiveImage from '../common/ResponsiveImage';

// headingComponent: nivel del nombre según la jerarquía de la página que lo
// usa — h3 bajo el h2 "Featured stays" de Home, h2 directo bajo el h1 de
// Search (sin nivel intermedio) — para no romper heading-order
const HotelCard = ({ hotel, headingComponent = 'h3' }) => {
  const rating = hotel.rating || 0;

  return (
    <Card
      sx={{
        position: 'relative',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        transition: 'box-shadow 200ms ease, transform 200ms ease',
        '@media (prefers-reduced-motion: no-preference)': {
          '&:hover': { transform: 'translateY(-3px)' },
        },
      }}
    >
      <Box sx={{ position: 'relative' }}>
        <ResponsiveImage src={hotelImage(hotel)} alt={hotel.name} aspectRatio="4 / 3" />
        <Box
          sx={{
            position: 'absolute',
            top: 12,
            right: 12,
            bgcolor: 'secondary.main',
            color: 'primary.main',
            px: 1.5,
            py: 0.5,
            borderRadius: 1,
          }}
        >
          <Typography variant="subtitle2" component="p" fontWeight={700}>
            {formatMajorAmount(hotel.price_per_night)}
            <Typography component="span" variant="caption" sx={{ ml: 0.5 }}>
              / night
            </Typography>
          </Typography>
        </Box>
      </Box>

      <CardContent sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', p: 2.5 }}>
        <Typography
          variant="h6"
          component={headingComponent}
          sx={{ fontWeight: 600, mb: 0.5, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
        >
          <Box
            component={Link}
            to={`/hotels/${hotel.id}`}
            // FE2: el ::after estira el área del link a la card entera, así
            // que hover/focus en cualquier punto precargan el chunk de detail
            onPointerEnter={prefetchHotelDetailChunk}
            onFocus={prefetchHotelDetailChunk}
            sx={{
              color: 'inherit',
              textDecoration: 'none',
              '&:hover': { color: 'primary.main' },
              '&::after': { content: '""', position: 'absolute', inset: 0 },
            }}
          >
            {hotel.name}
          </Box>
        </Typography>

        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1, color: 'text.secondary' }}>
          <LocationIcon sx={{ fontSize: 18, mr: 0.5 }} aria-hidden />
          <Typography variant="body2" noWrap>
            {hotel.city}, {hotel.country}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
          <Rating value={rating} precision={0.5} size="small" readOnly aria-hidden />
          <Typography variant="body2" color="text.secondary">
            {rating.toFixed(1)} out of 5
          </Typography>
        </Box>

        <Typography variant="body2" color="text.secondary" sx={{ mb: 2, lineHeight: 1.6 }}>
          {truncateText(hotel.description, 90)}
        </Typography>

        {hotel.amenities?.length > 0 && (
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mb: 1.5 }}>
            {hotel.amenities.slice(0, 3).map((amenity) => (
              <Chip
                key={amenity}
                size="small"
                label={amenityLabel(amenity)}
                sx={{ bgcolor: 'background.default', fontSize: '0.72rem', height: 24 }}
              />
            ))}
            {hotel.amenities.length > 3 && (
              <Chip
                size="small"
                label={`+${hotel.amenities.length - 3}`}
                sx={{ bgcolor: 'background.default', fontSize: '0.72rem', height: 24 }}
              />
            )}
          </Box>
        )}

        <Box sx={{ mt: 'auto', pt: 1.5, borderTop: '1px solid', borderColor: 'divider' }}>
          <Typography variant="caption" color="text.secondary">
            {hotel.available_rooms} room{hotel.available_rooms === 1 ? '' : 's'} in this property
          </Typography>
        </Box>
      </CardContent>
    </Card>
  );
};

export default HotelCard;
