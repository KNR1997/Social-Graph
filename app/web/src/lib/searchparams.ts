import { createSerializer, parseAsInteger, parseAsString } from 'nuqs';

export const searchParams = {
  page: parseAsInteger.withDefault(1),
  perPage: parseAsInteger.withDefault(10),
  name: parseAsString,
  gender: parseAsString,
  category: parseAsString,
  role: parseAsString,
  sort: parseAsString
  // advanced filter
  // filters: getFiltersStateParser().withDefault([]),
  // joinOperator: parseAsStringEnum(['and', 'or']).withDefault('and')
};

// The Next version also exported a `searchParamsCache` built with
// createSearchParamsCache, so server components could read the URL during
// render. There is no server render here -- the tables read the same params
// on the client with nuqs' useQueryStates -- so only the serializer remains.
export const serialize = createSerializer(searchParams);
