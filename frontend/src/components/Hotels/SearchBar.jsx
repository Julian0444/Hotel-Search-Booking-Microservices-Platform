/**
 * Search Bar Component
 * Hotel search input with submit functionality
 */

import { useState } from 'react';
import { Box, TextField, Button, InputAdornment, Paper } from '@mui/material';
import { Search as SearchIcon } from '@mui/icons-material';

const SearchBar = ({ onSearch, initialQuery = '', compact = false }) => {
  const [query, setQuery] = useState(initialQuery);

  const handleSubmit = (e) => {
    e.preventDefault();
    onSearch(query);
  };

  return (
    <Paper
      component="form"
      onSubmit={handleSubmit}
      elevation={compact ? 1 : 3}
      sx={{
        p: compact ? 1 : 2,
        display: 'flex',
        flexDirection: { xs: 'column', sm: 'row' },
        gap: 2,
        borderRadius: 2,
        bgcolor: 'white',
      }}
    >
      <TextField
        fullWidth
        placeholder="Search by city, country, or hotel name..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        variant="outlined"
        size={compact ? 'small' : 'medium'}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon color="action" />
            </InputAdornment>
          ),
        }}
        sx={{
          '& .MuiOutlinedInput-root': {
            bgcolor: 'background.default',
            '& fieldset': { borderColor: 'transparent' },
            '&:hover fieldset': { borderColor: 'primary.light' },
            '&.Mui-focused fieldset': { borderColor: 'primary.main' },
          },
        }}
      />
      <Button
        type="submit"
        variant="contained"
        size={compact ? 'medium' : 'large'}
        sx={{ minWidth: { xs: '100%', sm: 160 }, py: compact ? 1 : 1.5 }}
      >
        Search
      </Button>
    </Paper>
  );
};

export default SearchBar;
