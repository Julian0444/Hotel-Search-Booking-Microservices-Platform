/**
 * Gallery editorial (plan 13 fase 5): se adapta a 0/1/2/4+ imágenes sin
 * huecos — hero sola, par a dos columnas, mosaico con hero dominante.
 * "View all photos" abre un lightbox con teclado y focus trap (Dialog MUI).
 */

import { useState } from 'react';
import {
  Box,
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  ImageList,
  ImageListItem,
} from '@mui/material';
import { Close as CloseIcon, Collections as CollectionsIcon } from '@mui/icons-material';
import ResponsiveImage from '../common/ResponsiveImage';
import { HOTEL_FALLBACK_IMAGE } from '../../constants';

const HotelGallery = ({ hotel }) => {
  const [lightboxOpen, setLightboxOpen] = useState(false);
  const images = hotel.images?.length ? hotel.images : [HOTEL_FALLBACK_IMAGE];
  const count = images.length;

  const altFor = (index) => `${hotel.name} — photo ${index + 1} of ${count}`;

  return (
    <>
      <Box sx={{ position: 'relative' }}>
        {count === 1 && (
          <ResponsiveImage src={images[0]} alt={altFor(0)} aspectRatio="21 / 9" priority sx={{ borderRadius: 2 }} />
        )}

        {count === 2 && (
          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1 }}>
            {images.map((src, index) => (
              <ResponsiveImage
                key={src}
                src={src}
                alt={altFor(index)}
                aspectRatio="4 / 3"
                priority={index === 0}
                sx={{ borderRadius: 2 }}
              />
            ))}
          </Box>
        )}

        {count >= 3 && (
          <Box
            sx={{
              display: 'grid',
              gap: 1,
              gridTemplateColumns: { xs: '1fr', sm: '2fr 1fr' },
            }}
          >
            <ResponsiveImage src={images[0]} alt={altFor(0)} aspectRatio="4 / 3" priority sx={{ borderRadius: 2 }} />
            <Box sx={{ display: { xs: 'none', sm: 'grid' }, gap: 1, gridTemplateRows: '1fr 1fr' }}>
              {images.slice(1, 3).map((src, index) => (
                <ResponsiveImage
                  key={src}
                  src={src}
                  alt={altFor(index + 1)}
                  aspectRatio="4 / 3"
                  sx={{ borderRadius: 2, height: '100%' }}
                />
              ))}
            </Box>
          </Box>
        )}

        {count > 1 && (
          <Button
            size="small"
            variant="contained"
            color="inherit"
            startIcon={<CollectionsIcon />}
            onClick={() => setLightboxOpen(true)}
            sx={{
              position: 'absolute',
              bottom: 12,
              right: 12,
              bgcolor: 'background.paper',
              color: 'text.primary',
              '&:hover': { bgcolor: 'background.default' },
            }}
          >
            View all photos ({count})
          </Button>
        )}
      </Box>

      <Dialog
        open={lightboxOpen}
        onClose={() => setLightboxOpen(false)}
        fullWidth
        maxWidth="md"
        aria-labelledby="gallery-title"
      >
        <DialogTitle id="gallery-title" sx={{ pr: 6 }}>
          {hotel.name} — photos
          <IconButton
            aria-label="Close photo gallery"
            onClick={() => setLightboxOpen(false)}
            sx={{ position: 'absolute', right: 8, top: 10 }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent>
          <ImageList variant="masonry" cols={2} gap={8}>
            {images.map((src, index) => (
              <ImageListItem key={src}>
                <ResponsiveImage src={src} alt={altFor(index)} aspectRatio="4 / 3" sx={{ borderRadius: 1 }} />
              </ImageListItem>
            ))}
          </ImageList>
        </DialogContent>
      </Dialog>
    </>
  );
};

export default HotelGallery;
