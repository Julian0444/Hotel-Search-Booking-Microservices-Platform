/**
 * useHotelSearchParams (plan 13 fase 3, RV22/F13-03): la URL es la ÚNICA
 * fuente de verdad de q/page/sort — nada de duplicarla en useState. Copiar
 * la URL reproduce la pantalla; Back/Forward restauran todo.
 */

import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router';
import { DEFAULT_SORT, PAGINATION, SORT_OPTIONS } from '../constants';

const VALID_SORTS = new Set(SORT_OPTIONS.map((option) => option.value));

export const useHotelSearchParams = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const q = searchParams.get('q') || '';

  const rawPage = Number(searchParams.get('page'));
  const page = Number.isInteger(rawPage) && rawPage >= 1 ? rawPage : 1;

  const rawSort = searchParams.get('sort') || DEFAULT_SORT;
  const sort = VALID_SORTS.has(rawSort) ? rawSort : DEFAULT_SORT;

  const limit = PAGINATION.DEFAULT_PAGE_SIZE;
  const offset = (page - 1) * limit;

  const update = useCallback(
    (patch) => {
      setSearchParams(
        (previous) => {
          const next = new URLSearchParams(previous);
          const merged = {
            q: patch.q !== undefined ? patch.q : next.get('q') || '',
            page: patch.page !== undefined ? patch.page : Number(next.get('page')) || 1,
            sort: patch.sort !== undefined ? patch.sort : next.get('sort') || DEFAULT_SORT,
          };
          // Nueva búsqueda o nuevo sort resetean la página (F13-03)
          if (patch.q !== undefined || (patch.sort !== undefined && patch.page === undefined)) {
            merged.page = patch.page !== undefined ? patch.page : 1;
          }

          const params = new URLSearchParams();
          if (merged.q) params.set('q', merged.q);
          if (merged.page > 1) params.set('page', String(merged.page));
          if (merged.sort && merged.sort !== DEFAULT_SORT) params.set('sort', merged.sort);
          return params;
        },
        { replace: false },
      );
    },
    [setSearchParams],
  );

  return useMemo(
    () => ({
      q,
      page,
      sort,
      limit,
      offset,
      setQuery: (value) => update({ q: value }),
      setPage: (value) => update({ page: value }),
      setSort: (value) => update({ sort: value }),
    }),
    [q, page, sort, limit, offset, update],
  );
};
