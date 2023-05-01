import React, { Fragment, useState } from 'react';
import { Paper, Popover, Stack, TableCell, TableRow } from '@mui/material';
import { Player, Team } from '../api';
import Typography from '@mui/material/Typography';
import Grid2 from '@mui/material/Unstable_Grid2';

export interface TableRowContextMenuProps {
    player: Player;
}

export type ContextMenu = {
    mouseX: number;
    mouseY: number;
} | null;

export const TableRowContextMenu = ({
    player
}: TableRowContextMenuProps): JSX.Element => {
    const [anchorEl, setAnchorEl] = useState<HTMLTableRowElement | null>(null);
    //const [contextMenu, setContextMenu] = useState<ContextMenu>(null);
    const [open, setOpen] = useState(false);

    // const handleContextMenu = (
    //     event: React.MouseEvent<HTMLTableRowElement>
    // ) => {
    //     event.preventDefault();
    //     setContextMenu(
    //         contextMenu === null
    //             ? {
    //                   mouseX: event.clientX + 2,
    //                   mouseY: event.clientY - 6
    //               }
    //             : // repeated contextmenu when it is already open closes it with Chrome 84 on Ubuntu
    //               // Other native context menus might behave different.
    //               // With this behavior we prevent contextmenu from the backdrop to re-locale existing context menus.
    //               null
    //     );
    //     setAnchorEl(event.currentTarget);
    // };

    let bg = '';
    if (player.team == Team.BLU) {
        bg = '#0c1341';
    } else if (player.team == Team.RED) {
        bg = '#062c15';
    }
    let status = '';
    if (player.number_of_vac_bans) {
        status += `VAC: ${player.number_of_vac_bans}`;
        bg = '#383615';
    }
    if (player.match != null) {
        status = `${player.match.origin} [${player.match.attributes.join(
            ','
        )}]`;
        bg = '#500e0e';
    }
    const makeInfoRow = (key: string, value: string): JSX.Element[] => {
        return [
            <Grid2 xs={3} key={`${key}`} padding={0}>
                <Typography variant={'button'} textAlign={'right'}>
                    {key}
                </Typography>
            </Grid2>,
            <Grid2 xs={9} padding={0}>
                <Typography variant={'button'}>{value}</Typography>
            </Grid2>
        ];
    };
    return (
        <Fragment>
            <TableRow
                hover
                //onClick={handleClick}
                //onContextMenu={handleContextMenu}
                style={{ backgroundColor: bg }}
                key={`row-${player.steam_id}`}
                onMouseEnter={(
                    event: React.MouseEvent<HTMLTableRowElement>
                ) => {
                    setAnchorEl(event.currentTarget);
                    setOpen(true);
                }}
                onMouseLeave={() => {
                    setAnchorEl(null);
                    setOpen(false);
                }}
            >
                <TableCell style={{ padding: 3 }}>{player.name}</TableCell>
                <TableCell>{player.kills}</TableCell>
                <TableCell>{player.ping}</TableCell>
                <TableCell>{status}</TableCell>
            </TableRow>

            <Popover
                open={open}
                sx={{
                    pointerEvents: 'none'
                }}
                anchorEl={anchorEl}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'left'
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
                disablePortal={false}
                disableRestoreFocus
            >
                <Paper style={{ maxWidth: 650 }}>
                    <Stack padding={1} direction={'row'} spacing={1}>
                        <img
                            height={184}
                            width={184}
                            alt={player.name}
                            src={`https://avatars.cloudflare.steamstatic.com/${player.avatar_hash}_full.jpg`}
                        />
                        <Grid2 container>
                            {makeInfoRow('Name', player.name)}
                            {makeInfoRow('Kills', player.kills.toString())}
                            {makeInfoRow('Deaths', player.deaths.toString())}
                            {makeInfoRow('Ping', player.ping.toString())}
                            {makeInfoRow(
                                'Vac Bans',
                                player.number_of_vac_bans.toString()
                            )}
                            {makeInfoRow(
                                'Game Bans',
                                player.number_of_game_bans.toString()
                            )}
                            {player.match &&
                                makeInfoRow(
                                    'List(s)',
                                    `[${player.match.attributes.join(',')}] (${
                                        player.match.origin
                                    })`
                                )}
                        </Grid2>
                    </Stack>
                </Paper>
            </Popover>
        </Fragment>
    );
};
