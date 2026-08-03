/**
 * Fase 9 (FE2) — prefetch intencional: hover o focus sobre el link de la
 * card precargan el chunk de HotelDetail (y solo ese).
 */

import { describe, expect, it, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders } from '../../test/render';
import { hotelFixture } from '../../test/fixtures';
import HotelCard from './HotelCard';
import { prefetchHotelDetailChunk } from '../../utils/prefetch';

vi.mock('../../utils/prefetch', () => ({
  prefetchHotelDetailChunk: vi.fn(),
}));

describe('HotelCard prefetch', () => {
  it('hover sobre la card dispara el prefetch del chunk de detail', async () => {
    const { user } = renderWithProviders(<HotelCard hotel={hotelFixture} />);

    await user.hover(screen.getByRole('link', { name: hotelFixture.name }));
    expect(prefetchHotelDetailChunk).toHaveBeenCalled();
  });

  it('focus por teclado también lo dispara', async () => {
    const { user } = renderWithProviders(<HotelCard hotel={hotelFixture} />);

    await user.tab();
    expect(screen.getByRole('link', { name: hotelFixture.name })).toHaveFocus();
    expect(prefetchHotelDetailChunk).toHaveBeenCalled();
  });
});
