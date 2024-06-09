import { useContext, useMemo, useState, MouseEvent } from 'react';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Popover from '@mui/material/Popover';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableSortLabel from '@mui/material/TableSortLabel';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import ViewColumnIcon from '@mui/icons-material/ViewColumn';
import { Trans } from 'react-i18next';
import { PlayerTableContext } from '../context/PlayerTableContext';
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

export type Order = 'asc' | 'desc';

interface HeadCell {
    disablePadding: boolean;
    id: validColumns;
    label: string;
    numeric: boolean;
    tooltip: string;
}

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

const headCells: readonly HeadCell[] = [
    {
        id: 'user_id',
        numeric: true,
        disablePadding: false,
        label: 'uid',
        tooltip: 'Players in-Game user id'
    },
    {
        id: 'name',
        numeric: false,
        disablePadding: false,
        label: 'name',
        tooltip: 'Players current name, as reported by the game server'
    },
    {
        id: 'score',
        numeric: true,
        disablePadding: false,
        label: 'score',
        tooltip: 'Players current score'
    },
    {
        id: 'kills',
        numeric: true,
        disablePadding: false,
        label: 'kills',
        tooltip: 'Players current kills'
    },
    {
        id: 'deaths',
        numeric: true,
        disablePadding: false,
        label: 'deaths',
        tooltip: 'Players current deaths'
    },
    {
        id: 'kpm',
        numeric: true,
        disablePadding: false,
        label: 'kpm',
        tooltip:
            'Players kills per minute. Calculated from when you first see the player in the server, not how long they have actually been in the server'
    },
    {
        id: 'health',
        numeric: true,
        disablePadding: false,
        label: 'health',
        tooltip: 'Shows player current health'
    },
    {
        id: 'connected',
        numeric: true,
        disablePadding: false,
        label: 'time',
        tooltip: 'How long the player has been connected to the server'
    },
    {
        id: 'map_time',
        numeric: true,
        disablePadding: false,
        label: 'map time',
        tooltip: 'How long ist been since you first joined the map'
    },
    {
        id: 'ping',
        numeric: true,
        disablePadding: false,
        label: 'ping',
        tooltip: 'Players current latency'
    }
];

export const ColumnConfigButton = () => {
    const { saveSelectedColumns, enabledColumns } =
        useContext(PlayerTableContext);
    const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);

    const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const handleColumnsChange = (
        _: MouseEvent<HTMLElement>,
        newFormats: validColumns[]
    ) => {
        saveSelectedColumns(newFormats);
    };

    const open = Boolean(anchorEl);
    return (
        <>
            <IconButton onClick={handleClick}>
                <ViewColumnIcon color={'primary'} />
            </IconButton>
            <Popover
                open={open}
                onClose={handleClose}
                id={'column-config-popover'}
                anchorEl={anchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left'
                }}
            >
                <Paper>
                    <Stack>
                        <ToggleButtonGroup
                            color="primary"
                            orientation={'vertical'}
                            value={enabledColumns}
                            onChange={handleColumnsChange}
                            aria-label="Visible Columns"
                        >
                            {headCells.map((r) => {
                                return (
                                    <ToggleButton
                                        name={r.label}
                                        value={r.id}
                                        key={`column-toggle-${r.id}`}
                                    >
                                        <Trans
                                            i18nKey={`player_table.column.${r.id}`}
                                        />
                                    </ToggleButton>
                                );
                            })}
                        </ToggleButtonGroup>
                    </Stack>
                </Paper>
            </Popover>
        </>
    );
};

const PlayerTableHead = <T,>({ table }: { table: TSTable<T> }) => {
    const order = table.getState().sorting[0];
    return (
        <TableHead>
            {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                    {headerGroup.headers.map((header) => (
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
        state: {
            sorting,
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
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('personaname', {
            header: () => <TableHeading>Name</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('score', {
            header: () => <TableHeading>Score</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('kills', {
            header: () => <TableHeading>K</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('deaths', {
            header: () => <TableHeading>D</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('kpm', {
            header: () => <TableHeading>KPM</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('health', {
            header: () => <TableHeading>HP</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('connected', {
            header: () => <TableHeading>Conn.</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('map_time', {
            header: () => <TableHeading>Map Time</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        }),
        columnHelper.accessor('ping', {
            header: () => <TableHeading>Ping</TableHeading>,
            cell: (info) => <TableCell>{`#${info.getValue()}`}</TableCell>
        })
    ];
};
