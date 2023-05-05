import React, { useMemo } from 'react';

import {
    Box,
    IconButton,
    Paper,
    Popover,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    TableSortLabel
} from '@mui/material';
import { Player } from '../api';
import { TableRowContextMenu } from './TableRowContextMenu';
import { visuallyHidden } from '@mui/utils';
import FilterListOutlinedIcon from '@mui/icons-material/FilterListOutlined';

export interface PlayerTableProps {
    onRequestSort: (
        event: React.MouseEvent<unknown>,
        property: keyof Player
    ) => void;
    order: Order;
    orderBy: string;
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
    id: keyof Player;
    label: string;
    numeric: boolean;
}

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

export const ColumnConfigButton = () => {
    const [anchorEl, setAnchorEl] = React.useState<HTMLButtonElement | null>(
        null
    );

    const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const open = Boolean(anchorEl);
    return (
        <>
            <IconButton onClick={handleClick}>
                <FilterListOutlinedIcon color={'primary'} />
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
                    <Stack spacing={1}></Stack>
                </Paper>
            </Popover>
        </>
    );
};

const PlayerTableHead = (props: PlayerTableProps) => {
    const { order, orderBy, onRequestSort } = props;
    const createSortHandler =
        (property: keyof Player) => (event: React.MouseEvent<unknown>) => {
            onRequestSort(event, property);
        };

    return (
        <TableHead>
            <TableRow>
                {headCells.map((headCell) => (
                    <TableCell
                        key={headCell.id}
                        align={headCell.numeric ? 'right' : 'left'}
                        padding={headCell.disablePadding ? 'none' : 'normal'}
                        sortDirection={orderBy === headCell.id ? order : false}
                    >
                        <TableSortLabel
                            active={orderBy === headCell.id}
                            direction={orderBy === headCell.id ? order : 'asc'}
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

interface PlayerTableRootProps {
    players: Player[];
    matchesOnly?: boolean;
}

export const PlayerTable = ({ players, matchesOnly }: PlayerTableRootProps) => {
    const [order, setOrder] = React.useState<Order>('desc');
    const [orderBy, setOrderBy] = React.useState<keyof Player>('name');

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

    return (
        <Paper sx={{ width: '100%', overflow: 'hidden' }}>
            <TableContainer>
                <Table
                    stickyHeader
                    aria-label="sticky table"
                    size="small"
                    padding={'none'}
                    sx={{
                        '& .MuiTableRow-root:hover': {
                            backgroundColor: 'primary.light'
                        }
                    }}
                >
                    <PlayerTableHead
                        players={players}
                        order={order}
                        orderBy={orderBy}
                        onRequestSort={handleRequestSort}
                    />

                    <TableBody>
                        {visibleRows.map((player, i) => (
                            <TableRowContextMenu
                                player={player}
                                key={`row-${i}-${player.steam_id}`}
                            />
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>
        </Paper>
    );
};
