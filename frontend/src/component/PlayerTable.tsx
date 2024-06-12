import { useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import { Trans } from 'react-i18next';
import { useGameState } from '../context/GameStateContext.ts';
import { Player } from '../api.ts';
import {
    ColumnFiltersState,
    createColumnHelper,
    flexRender,
    getCoreRowModel,
    getFilteredRowModel,
    getSortedRowModel,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { TableHeading } from './TableHeading.tsx';
import { Table as TSTable } from '@tanstack/react-table';
import { intervalToDuration } from 'date-fns/intervalToDuration';
import { humanDuration } from '../util.ts';

export type Order = 'asc' | 'desc';

export type validColumns =
    | 'user_id'
    | 'name'
    | 'score'
    | 'kills'
    | 'deaths'
    | 'kpm'
    | 'connected'
    | 'map_time'
    | 'ping'
    | 'health'
    | 'alive';

const PlayerTableHead = <T,>({ table }: { table: TSTable<T> }) => {
    const order = table.getState().sorting[0];
    return (
        <TableHead>
            {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                    {table.getVisibleLeafColumns().map((header) => (
                        <TableCell key={header.id}>
                            <TableSortLabel
                                active={order.id === header.id}
                                direction={order.desc ? 'desc' : 'asc'}
                            >
                                <Trans
                                    i18nKey={`player_table.column.${header.id}`}
                                />
                                {order.id === header.id ? (
                                    <Box
                                        component="span"
                                        sx={{ display: 'none' }}
                                    >
                                        {order.desc
                                            ? 'sorted descending'
                                            : 'sorted ascending'}
                                    </Box>
                                ) : null}
                            </TableSortLabel>
                        </TableCell>
                    ))}
                </TableRow>
            ))}
        </TableHead>
    );
};

export const PlayerTable = () => {
    const [sorting, setSorting] = useState<SortingState>([
        { id: 'kills', desc: true }
    ]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
    const [columnVisibility, setColumnVisibility] = useState({});
    const { state } = useGameState();

    // const whitelist = useMutation(addWhitelistMutation());
    //
    // const onWhitelist = useCallback(
    //     async (steamId: string) => {
    //         whitelist.mutate({ steamId });
    //     },
    //     [whitelist.mutate]
    // );

    const columns = useMemo(makeColumns, []);

    const table = useReactTable<Player>({
        data: state.players,
        columns: columns,
        autoResetPageIndex: true,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: columnFilters ? getFilteredRowModel() : undefined,
        getSortedRowModel: sorting ? getSortedRowModel() : undefined,
        onColumnFiltersChange: setColumnFilters,
        onSortingChange: setSorting,
        onColumnVisibilityChange: setColumnVisibility,
        state: {
            sorting,
            columnVisibility,
            columnFilters
        }
    });

    return (
        <TableContainer sx={{ overflow: 'hidden' }}>
            <Table aria-label="Player table" size="small" padding={'none'}>
                <PlayerTableHead table={table} />
                <TableBody>
                    {table.getRowModel().rows.map((row) => (
                        <TableRow key={row.id} hover>
                            {row.getVisibleCells().map((cell) => (
                                <TableCell key={cell.id}>
                                    {flexRender(
                                        cell.column.columnDef.cell,
                                        cell.getContext()
                                    )}
                                </TableCell>
                            ))}
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </TableContainer>
    );
};

const columnHelper = createColumnHelper<Player>();

const makeColumns = () => {
    return [
        columnHelper.accessor('user_id', {
            header: () => <TableHeading>ID</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('personaname', {
            header: () => <TableHeading>Name</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('score', {
            header: () => <TableHeading>Score</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('kills', {
            header: () => <TableHeading>K</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('deaths', {
            header: () => <TableHeading>D</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('kpm', {
            header: () => <TableHeading>KPM</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('health', {
            header: () => <TableHeading>HP</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('connected', {
            header: () => <TableHeading>Conn.</TableHeading>,
            cell: (info) => (
                <TableCell>{`${humanDuration(intervalToDuration({ start: 0, end: info.getValue() / 1000 / 1000 / 1000 }))}`}</TableCell>
            )
        }),
        columnHelper.accessor('map_time', {
            header: () => <TableHeading>Map Time</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('ping', {
            header: () => <TableHeading>Ping</TableHeading>,
            cell: (info) => <TableCell>{`${info.getValue()}`}</TableCell>
        })
    ];
};
