/**
 * Skeleton del grid de resultados (plan 13 fase 3): solo primera carga;
 * al paginar el grid anterior queda visible con keepPreviousData.
 */

import { Card, CardContent, Grid, Skeleton } from '@mui/material';

const HotelGridSkeleton = ({ count = 6 }) => (
  <Grid container spacing={3} aria-hidden>
    {Array.from({ length: count }).map((_, index) => (
      <Grid key={index} size={{ xs: 12, sm: 6, md: 4 }}>
        <Card>
          <Skeleton variant="rectangular" sx={{ aspectRatio: '4 / 3', height: 'auto' }} />
          <CardContent>
            <Skeleton variant="text" height={30} width="75%" />
            <Skeleton variant="text" height={20} width="50%" />
            <Skeleton variant="text" height={20} width="35%" />
          </CardContent>
        </Card>
      </Grid>
    ))}
  </Grid>
);

export default HotelGridSkeleton;
