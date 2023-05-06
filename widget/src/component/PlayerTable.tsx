import React, { useCallback, useMemo, useState } from 'react';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
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
import Tooltip from '@mui/material/Tooltip';
import visuallyHidden from '@mui/utils/visuallyHidden';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';
import Typography from '@mui/material/Typography';
import { TableRowContextMenu } from './TableRowContextMenu';
import { Player, usePlayers } from '../api';

export interface PlayerTableProps {
    onRequestSort: (
        event: React.MouseEvent<unknown>,
        property: keyof Player
    ) => void;
    order: Order;
    orderBy: string;
    enabledColumns: validColumns[];
    readonly players?: Player[];
    readonly matchesOnly?: boolean;
}

const descendingComparator = <T extends any>(a: T, b: T, orderBy: keyof T) => {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
};

type Order = 'asc' | 'desc';

const getComparator = <Key extends keyof any>(
    order: Order,
    orderBy: Key
): ((a: { [key in Key]: any }, b: { [key in Key]: any }) => number) =>
    order === 'asc'
        ? (a, b) => descendingComparator(a, b, orderBy)
        : (a, b) => -descendingComparator(a, b, orderBy);

const stableSort = <T extends any>(
    array: readonly T[],
    comparator: (a: T, b: T) => number
) => {
    const stabilizedThis = array.map((el, index) => [el, index] as [T, number]);
    stabilizedThis.sort((a, b) => {
        const order = comparator(a[0], b[0]);
        if (order !== 0) {
            return order;
        }
        return a[1] - b[1];
    });
    return stabilizedThis.map((el) => el[0]);
};

interface HeadCell {
    disablePadding: boolean;
    id: validColumns;
    label: string;
    numeric: boolean;
}

export type validColumns =
    | 'user_id'
    | 'name'
    | 'kills'
    | 'deaths'
    | 'connected'
    | 'ping';

const headCells: readonly HeadCell[] = [
    {
        id: 'user_id',
        numeric: true,
        disablePadding: false,
        label: 'uid'
    },
    {
        id: 'name',
        numeric: false,
        disablePadding: false,
        label: 'name'
    },
    {
        id: 'kills',
        numeric: true,
        disablePadding: false,
        label: 'kills'
    },
    {
        id: 'deaths',
        numeric: true,
        disablePadding: false,
        label: 'deaths'
    },
    {
        id: 'connected',
        numeric: true,
        disablePadding: false,
        label: 'time'
    },
    {
        id: 'ping',
        numeric: true,
        disablePadding: false,
        label: 'ping'
    }
];

interface ColumnConfigButtonProps {
    enabledColumns: validColumns[];
    setEnabledColumns: (columns: validColumns[]) => void;
}

export const ColumnConfigButton = ({
    setEnabledColumns,
    enabledColumns
}: ColumnConfigButtonProps) => {
    const [anchorEl, setAnchorEl] = React.useState<HTMLButtonElement | null>(
        null
    );

    const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const handleColumnsChange = (
        _: React.MouseEvent<HTMLElement>,
        newFormats: validColumns[]
    ) => {
        setEnabledColumns(newFormats);
    };

    const open = Boolean(anchorEl);
    return (
        <>
            <IconButton onClick={handleClick}>
                <SettingsOutlinedIcon color={'primary'} />
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
                                        {r.label}
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

const PlayerTableHead = ({
    order,
    orderBy,
    onRequestSort,
    enabledColumns
}: PlayerTableProps) => {
    const createSortHandler =
        (property: keyof Player) => (event: React.MouseEvent<unknown>) => {
            onRequestSort(event, property);
        };

    return (
        <TableHead>
            <TableRow>
                {headCells
                    .filter(
                        (c) => enabledColumns.includes(c.id) || !enabledColumns
                    )
                    .map((headCell) => (
                        <TableCell
                            key={headCell.id}
                            align={headCell.numeric ? 'right' : 'left'}
                            padding={
                                headCell.disablePadding ? 'none' : 'normal'
                            }
                            sortDirection={
                                orderBy === headCell.id ? order : false
                            }
                        >
                            <TableSortLabel
                                active={orderBy === headCell.id}
                                direction={
                                    orderBy === headCell.id ? order : 'asc'
                                }
                                onClick={createSortHandler(headCell.id)}
                            >
                                {headCell.label}
                                {orderBy === headCell.id ? (
                                    <Box component="span" sx={visuallyHidden}>
                                        {order === 'desc'
                                            ? 'sorted descending'
                                            : 'sorted ascending'}
                                    </Box>
                                ) : null}
                            </TableSortLabel>
                        </TableCell>
                    ))}
            </TableRow>
        </TableHead>
    );
};

interface PlayerTableRootProps {}

const getDefaultColumns = (): validColumns[] => {
    const defaultCols: validColumns[] = [
        'user_id',
        'name',
        'kills',
        'deaths',
        'connected',
        'ping'
    ];

    let val = localStorage.getItem('enabledColumns');
    if (!val) {
        return defaultCols;
    }
    try {
        const cols = JSON.parse(val);
        if (!cols) {
            return defaultCols;
        }
        return cols;
    } catch (_) {
        return defaultCols;
    }
};

export const PlayerTable = ({}: PlayerTableRootProps) => {
    const [order, setOrder] = React.useState<Order>('desc');
    const [orderBy, setOrderBy] = React.useState<keyof Player>('name');
    const [matchesOnly, setMatchesOnly] = useState(
        // Surely strings are the only types
        JSON.parse(localStorage.getItem('matchesOnly') || 'false') === true
    );
    const [enabledColumns, setEnabledColumns] = useState<validColumns[]>(
        getDefaultColumns()
    );

    const players = usePlayers();

    const handleRequestSort = (
        _: React.MouseEvent<unknown>,
        property: keyof Player
    ) => {
        const isAsc = orderBy === property && order === 'asc';
        setOrder(isAsc ? 'desc' : 'asc');
        setOrderBy(property);
    };

    // exampleArray.slice().sort(exampleComparator)
    const visibleRows = useMemo(
        () =>
            stableSort(
                players.filter((p) => !matchesOnly || p.matches.length),
                getComparator(order, orderBy)
            ),
        [order, orderBy, players, matchesOnly]
    );

    const updateSelectedColumns = useCallback(
        (columns: validColumns[]) => {
            setEnabledColumns(columns);
            localStorage.setItem('enabledColumns', JSON.stringify(columns));
        },
        [setEnabledColumns]
    );

    return (
        <Paper sx={{ width: '100%', overflow: 'hidden' }}>
            <Stack>
                <Stack direction={'row'}>
                    <ButtonGroup>
                        <Tooltip
                            title={
                                'Show only players with some sort of negative status'
                            }
                        >
                            <IconButton
                                onClick={() => {
                                    setMatchesOnly((prevState) => {
                                        localStorage.setItem(
                                            'matchesOnly',
                                            `${!prevState}`
                                        );
                                        return !prevState;
                                    });
                                }}
                            >
                                <FilterListOutlinedIcon color={'primary'} />
                            </IconButton>
                        </Tooltip>
                        <ColumnConfigButton
                            enabledColumns={enabledColumns}
                            setEnabledColumns={updateSelectedColumns}
                        />
                    </ButtonGroup>
                    <Box sx={{ display: 'flex', alignItems: 'center' }}>
                        <Typography variant={'overline'}></Typography>
                    </Box>
                </Stack>
                <TableContainer>
                    <Table
                        stickyHeader
                        aria-label="sticky table"
                        size="small"
                        padding={'none'}
                    >
                        <PlayerTableHead
                            players={players}
                            order={order}
                            orderBy={orderBy}
                            onRequestSort={handleRequestSort}
                            enabledColumns={enabledColumns}
                        />

                        <TableBody>
                            {visibleRows.map((player, i) => (
                                <TableRowContextMenu
                                    enabledColumns={enabledColumns}
                                    player={player}
                                    key={`row-${i}-${player.steam_id}`}
                                />
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            </Stack>
        </Paper>
    );
};
