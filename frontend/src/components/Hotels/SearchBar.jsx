/**
 * Search Bar (plan 13 fase 3): input etiquetado + submit. Sincroniza el
 * valor cuando la URL cambia desde afuera (Back/Forward, chip de clear).
 */

import { useEffect, useState } from 'react';
import { Paper, TextField, Button, InputAdornment } from '@mui/material';
import { Search as SearchIcon } from '@mui/icons-material';

const SearchBar = ({ onSearch, initialQuery = '', compact = false }) => {
  const [query, setQuery] = useState(initialQuery);

  useEffect(() => {
    setQuery(initialQuery);
  }, [initialQuery]);

  const handleSubmit = (event) => {
    event.preventDefault();
    onSearch(query.trim());
  };

  return (
    <Paper
      component="form"
      role="search"
      onSubmit={handleSubmit}
      elevation={compact ? 1 : 3}
      sx={{
        p: compact ? 1 : 1.5,
        display: 'flex',
        flexDirection: { xs: 'column', sm: 'row' },
        gap: 1.5,
        borderRadius: 2,
      }}
    >
      <TextField
        fullWidth
        label="Search stays"
        placeholder="City, country, or hotel name"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        variant="outlined"
        size={compact ? 'small' : 'medium'}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon color="action" aria-hidden />
            </InputAdornment>
          ),
        }}
      />
      <Button
        type="submit"
        variant="contained"
        size={compact ? 'medium' : 'large'}
        sx={{ minWidth: { xs: '100%', sm: 140 }, minHeight: 44 }}
      >
        Search
      </Button>
    </Paper>
  );
};

export default SearchBar;
