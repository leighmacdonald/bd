import { PlayerTable } from '../component/PlayerTable.tsx';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
    component: Index
});

function Index() {
    // const [order, setOrder] = useState<Order>(
    //     (localStorage.getItem('sortOrder') as Order) ?? 'desc'
    // );
    // const [orderBy, setOrderBy] = useState<validColumns>(
    //     (localStorage.getItem('sortBy') as validColumns) ?? 'personaname'
    // );
    // const [matchesOnly, setMatchesOnly] = useState(
    //     JSON.parse(localStorage.getItem('matchesOnly') || 'false') === true
    // );
    // const [enabledColumns, setEnabledColumns] =
    //     useState<validColumns[]>(getDefaultColumns());
    //
    // const saveSelectedColumns = useCallback(
    //     (columns: validColumns[]) => {
    //         setEnabledColumns(columns);
    //         localStorage.setItem('enabledColumns', JSON.stringify(columns));
    //     },
    //     [setEnabledColumns]
    // );
    //
    // const saveSortColumn = useCallback(
    //     (property: validColumns) => {
    //         const isAsc = orderBy === property && order === 'asc';
    //         const newOrder = isAsc ? 'desc' : 'asc';
    //         setOrder(newOrder);
    //         setOrderBy(property);
    //         localStorage.setItem('sortOrder', newOrder);
    //         localStorage.setItem('sortBy', property);
    //     },
    //     [order, orderBy]
    // );

    return <PlayerTable />;
}
