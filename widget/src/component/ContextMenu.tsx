import React, { Fragment, useState } from 'react';
import { Paper, Popover, Stack, TableCell, TableRow } from '@mui/material';
import { formatSeconds, Player, Team } from '../api';
import Typography from '@mui/material/Typography';
import Grid2 from '@mui/material/Unstable_Grid2';

export interface TableRowContextMenuProps {
    player: Player;
}

export const renderStatus = (player: Player): string => {
    let status = '';
    if (player.number_of_vac_bans) {
        status += `VAC: ${player.number_of_vac_bans}`;
    }
    if (player.match != null) {
        status = `${player.match.origin} [${player.match.attributes.join(
            ','
        )}]`;
    }
    return status;
};

interface userTheme {
    connectingBg: string;
    matchCheaterBg: string;
    matchBotBg: string;
    matchOtherBg: string;
    vacBansBg: string;
    gameBansBg: string;
    teamABg: string;
    teamBBg: string;
}

const createUserTheme = (): userTheme => {
    // TODO user configurable
    return {
        connectingBg: '#032a23',
        teamABg: '#062c15',
        matchBotBg: '#901380',
        matchCheaterBg: '#500e0e',
        matchOtherBg: '#0c1341',
        teamBBg: '#032a23',
        gameBansBg: '#383615',
        vacBansBg: '#55521f'
    };
};

const curTheme = createUserTheme();

export const rowColour = (player: Player): string => {
    if (player.match != null) {
        if (player.match.attributes.includes('cheater')) {
            return curTheme.matchCheaterBg;
        } else if (player.match.attributes.includes('bot')) {
            return curTheme.matchBotBg;
        }
        return curTheme.matchOtherBg;
    } else if (player.number_of_vac_bans) {
        return curTheme.vacBansBg;
    } else if (player.number_of_game_bans) {
        return curTheme.gameBansBg;
    } else if (player.team == Team.RED) {
        return curTheme.teamABg;
    } else if (player.team == Team.BLU) {
        return curTheme.teamBBg;
    }
    return curTheme.connectingBg;
};

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

    const makeInfoRow = (key: string, value: any): JSX.Element[] => {
        return [
            <Grid2 xs={3} key={`${key}-key`} padding={0}>
                <Typography variant={'button'} textAlign={'right'}>
                    {key}
                </Typography>
            </Grid2>,
            <Grid2 xs={9} key={`${key}-val`} padding={0}>
                <Typography variant={'button'}>{value}</Typography>
            </Grid2>
        ];
    };

    return (
        <Fragment key={`${player.steam_id}`}>
            <TableRow
                hover
                //onClick={handleClick}
                //onContextMenu={handleContextMenu}
                style={{ backgroundColor: rowColour(player) }}
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
                <TableCell align={'right'} style={{ paddingRight: 6 }}>
                    <Typography variant={'overline'}>
                        {player.user_id}
                    </Typography>
                </TableCell>
                <TableCell>
                    <Typography
                        sx={{ fontFamily: 'Monospace' }}
                        overflow={'ellipsis'}
                    >
                        {player.name}
                    </Typography>
                </TableCell>
                <TableCell align={'right'}>
                    <Typography variant={'overline'}>{player.kills}</Typography>
                </TableCell>
                <TableCell align={'right'}>
                    <Typography variant={'overline'}>
                        {player.deaths}
                    </Typography>
                </TableCell>
                <TableCell align={'right'}>
                    <Typography variant={'overline'}>
                        {formatSeconds(player.connected)}
                    </Typography>
                </TableCell>
                <TableCell align={'right'} style={{ paddingRight: 6 }}>
                    <Typography variant={'overline'}> {player.ping}</Typography>
                </TableCell>
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
                            {...makeInfoRow('UID', player.user_id)}
                            {...makeInfoRow('Name', player.name)}
                            {...makeInfoRow('Kills', player.kills.toString())}
                            {...makeInfoRow('Deaths', player.deaths.toString())}
                            {...makeInfoRow('Time', player.connected)}
                            {...makeInfoRow('Ping', player.ping.toString())}
                            {...makeInfoRow(
                                'Vac Bans',
                                player.number_of_vac_bans.toString()
                            )}
                            {...makeInfoRow(
                                'Game Bans',
                                player.number_of_game_bans.toString()
                            )}
                        </Grid2>
                    </Stack>
                </Paper>
            </Popover>
        </Fragment>
    );
};
