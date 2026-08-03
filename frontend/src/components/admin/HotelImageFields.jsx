/**
 * URLs de imagen del HotelForm (plan 13 fase 7): solo http(s) válidas,
 * preview con fallback y reordenamiento (la primera es la portada).
 */

import { useState } from 'react';
import { Box, Button, IconButton, TextField, Typography } from '@mui/material';
import {
  Add as AddIcon,
  Close as CloseIcon,
  ArrowUpward as UpIcon,
  ArrowDownward as DownIcon,
} from '@mui/icons-material';
import ResponsiveImage from '../common/ResponsiveImage';
import { isHttpUrl } from '../../utils/validators';

const HotelImageFields = ({ images, onChange }) => {
  const [input, setInput] = useState('');
  const [inputError, setInputError] = useState(null);

  const addImage = () => {
    const url = input.trim();
    if (!url) return;
    if (!isHttpUrl(url)) {
      setInputError('Image URLs must start with http:// or https://');
      return;
    }
    if (images.includes(url)) {
      setInputError('That URL is already in the list');
      return;
    }
    onChange([...images, url]);
    setInput('');
    setInputError(null);
  };

  const move = (index, delta) => {
    const next = [...images];
    const [item] = next.splice(index, 1);
    next.splice(index + delta, 0, item);
    onChange(next);
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start', mb: 2 }}>
        <TextField
          fullWidth
          size="small"
          label="Image URL"
          placeholder="https://…"
          value={input}
          onChange={(event) => {
            setInput(event.target.value);
            setInputError(null);
          }}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault();
              addImage();
            }
          }}
          error={!!inputError}
          helperText={inputError}
        />
        <Button onClick={addImage} variant="outlined" startIcon={<AddIcon />} sx={{ flexShrink: 0 }}>
          Add
        </Button>
      </Box>

      {images.length > 0 && (
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr 1fr', sm: 'repeat(4, 1fr)' }, gap: 1.5 }}>
          {images.map((url, index) => (
            <Box key={url} sx={{ position: 'relative' }}>
              <ResponsiveImage src={url} alt={`Hotel image ${index + 1}`} aspectRatio="4 / 3" sx={{ borderRadius: 1 }} />
              {index === 0 && (
                <Typography
                  variant="caption"
                  sx={{
                    position: 'absolute',
                    bottom: 4,
                    left: 4,
                    bgcolor: 'rgba(0,0,0,0.6)',
                    color: 'white',
                    px: 0.75,
                    borderRadius: 0.5,
                  }}
                >
                  Cover
                </Typography>
              )}
              <Box sx={{ position: 'absolute', top: 4, right: 4, display: 'flex', gap: 0.25 }}>
                {index > 0 && (
                  <IconButton size="small" aria-label={`Move image ${index + 1} earlier`} onClick={() => move(index, -1)} sx={{ bgcolor: 'rgba(255,255,255,0.85)' }}>
                    <UpIcon fontSize="inherit" />
                  </IconButton>
                )}
                {index < images.length - 1 && (
                  <IconButton size="small" aria-label={`Move image ${index + 1} later`} onClick={() => move(index, 1)} sx={{ bgcolor: 'rgba(255,255,255,0.85)' }}>
                    <DownIcon fontSize="inherit" />
                  </IconButton>
                )}
                <IconButton
                  size="small"
                  aria-label={`Remove image ${index + 1}`}
                  onClick={() => onChange(images.filter((i) => i !== url))}
                  sx={{ bgcolor: 'rgba(255,255,255,0.85)' }}
                >
                  <CloseIcon fontSize="inherit" />
                </IconButton>
              </Box>
            </Box>
          ))}
        </Box>
      )}
    </Box>
  );
};

export default HotelImageFields;
