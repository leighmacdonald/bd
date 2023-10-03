import React, { Fragment, useContext } from 'react';
import Paper from '@mui/material/Paper';
import Popover from '@mui/material/Popover';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import {
    avatarURL,
    formatSeconds,
    Player,
    Team,
    visibilityString
} from '../api';
import sb from '../img/sb.png';
import dead from '../img/dead.png';
import vac from '../img/vac.png';
import notes from '../img/notes.png';
import marked from '../img/marked.png';
import whitelist from '../img/whitelist.png';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableBody from '@mui/material/TableBody';
import { format, parseJSON } from 'date-fns';
import { TextareaAutosize } from '@mui/material';
import Table from '@mui/material/Table';
import { Trans, useTranslation } from 'react-i18next';
import { PlayerContextMenu } from './menu/PlayerContextMenu';
import { NullablePosition } from './menu/common';
import { PlayerTableContext } from '../context/PlayerTableContext';
import { SettingsContext } from '../context/SettingsContext';

export interface TableRowContextMenuProps {
    player: Player;
    onWhitelist: (steamId: string) => void;
}

export const bluColour = 'rgba(0,18,45,0.82)';
export const redColour = 'rgba(44,0,10,0.81)';

interface userTheme {
    disconnected: string;
    connectingBg: string;
    matchCheaterBg: string;
    matchBotBg: string;
    matchOtherBg: string;
    teamABg: string;
    teamBBg: string;
}

const createUserTheme = (): userTheme => {
    // TODO user configurable
    return {
        disconnected: '#2d2d2d',
        connectingBg: '#032a23',
        teamABg: bluColour,
        matchBotBg: '#901380',
        matchCheaterBg: '#500e0e',
        matchOtherBg: '#0c1341',
        teamBBg: redColour
    };
};

const curTheme = createUserTheme();

export const rowColour = (player: Player): string => {
    if (!player.is_connected) {
        return curTheme.disconnected;
    } else if (player.matches && player.matches.length) {
        if (
            player.matches.filter((m) => {
                m.attributes.includes('cheater');
            })
        ) {
            return curTheme.matchCheaterBg;
        } else if (
            player.matches.filter((m) => {
                m.attributes.includes('bot');
            })
        ) {
            return curTheme.matchBotBg;
        }
        return curTheme.matchOtherBg;
    } else if (player.team == Team.RED) {
        return curTheme.teamABg;
    } else if (player.team == Team.BLU) {
        return curTheme.teamBBg;
    }
    return curTheme.connectingBg;
};

export const PlayerTableRow = ({
    player
}: TableRowContextMenuProps): JSX.Element => {
    const { t } = useTranslation();

    const [hoverMenuPos, setHoverMenuPos] =
        React.useState<NullablePosition>(null);

    const [contextMenuPos, setContextMenuPos] =
        React.useState<NullablePosition>(null);

    const { settings, loading } = useContext(SettingsContext);
    const { enabledColumns } = useContext(PlayerTableContext);

    const handleMenuClose = () => {
        setContextMenuPos(null);
    };

    const handleRowClick = (event: React.MouseEvent<HTMLTableRowElement>) => {
        setContextMenuPos(
            contextMenuPos === null
                ? {
                      mouseX: event.clientX + 2,
                      mouseY: event.clientY - 6
                  }
                : // repeated contextmenu when it is already open closes it with Chrome 84 on Ubuntu
                  // Other native context menus might behave different.
                  // With this behavior we prevent contextmenu from the backdrop to re-locale existing context menus.
                  null
        );
    };

    const mouseEnter = (event: React.MouseEvent<HTMLTableRowElement>) => {
        setHoverMenuPos(
            contextMenuPos === null
                ? {
                      mouseX: event.clientX + 2,
                      mouseY: event.clientY - 6
                  }
                : null
        );
    };

    const mouseLeave = () => {
        setHoverMenuPos(null);
    };

    if (loading || !settings) {
        return <></>;
    }
    return (
        <Fragment key={`${player.steam_id}`}>
            <TableRow
                hover
                style={{
                    backgroundColor: rowColour(player),
                    cursor: 'pointer'
                }}
                key={`row-${player.steam_id}`}
                onMouseEnter={mouseEnter}
                onMouseLeave={mouseLeave}
                onClick={handleRowClick}
                sx={{
                    '&:hover': {
                        backgroundColor: 'primary'
                    }
                }}
            >
                {enabledColumns.includes('user_id') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.user_id}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('name') && (
                    <TableCell>
                        <Grid container spacing={1}>
                            {!player.alive && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={dead}
                                        alt={t('player_table.row.icon_dead')}
                                    />
                                </Grid>
                            )}

                            <Grid xs textOverflow={'clip'} overflow={'hidden'}>
                                <Typography
                                    overflow={'clip'}
                                    sx={{
                                        fontFamily: 'Monospace',
                                        maxWidth: '250px'
                                    }}
                                    textOverflow={'clip'}
                                    variant={'subtitle1'}
                                >
                                    {player.name}
                                </Typography>
                            </Grid>

                            {player.number_of_vac_bans > 0 && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={vac}
                                        alt={`${player.number_of_vac_bans} ${t(
                                            'player_table.row.vac_bans'
                                        )}`}
                                    />
                                </Grid>
                            )}
                            {player.sourcebans &&
                                player.sourcebans.length > 0 && (
                                    <Grid
                                        xs={'auto'}
                                        display="flex"
                                        justifyContent="center"
                                        alignItems="center"
                                    >
                                        <img
                                            width={18}
                                            height={18}
                                            src={sb}
                                            alt={`${
                                                player.sourcebans.length
                                            } ${t(
                                                'player_table.row.source_bans'
                                            )}`}
                                        />
                                    </Grid>
                                )}
                            {player.matches && player.matches?.length > 0 && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={marked}
                                        alt={t(
                                            'player_table.row.player_on_lists'
                                        )}
                                    />
                                </Grid>
                            )}
                            {player.whitelisted && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={whitelist}
                                        alt={t(
                                            'player_table.row.player_on_lists_whitelisted'
                                        )}
                                    />
                                </Grid>
                            )}
                            {player.notes.length > 0 && (
                                <Grid
                                    xs={'auto'}
                                    display="flex"
                                    justifyContent="center"
                                    alignItems="center"
                                >
                                    <img
                                        width={18}
                                        height={18}
                                        src={notes}
                                        alt={t('player_table.row.player_notes')}
                                    />
                                </Grid>
                            )}
                        </Grid>
                    </TableCell>
                )}
                {enabledColumns.includes('score') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.score}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('kills') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.kills}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('deaths') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.deaths}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('kpm') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.kpm.toPrecision(2)}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('health') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {player.alive ? player.health : 0}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('connected') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {formatSeconds(player.connected)}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('map_time') && (
                    <TableCell align={'right'}>
                        <Typography variant={'overline'}>
                            {formatSeconds(player.map_time)}
                        </Typography>
                    </TableCell>
                )}
                {enabledColumns.includes('ping') && (
                    <TableCell align={'right'} style={{ paddingRight: 6 }}>
                        <Typography variant={'overline'}>
                            {player.ping}
                        </Typography>
                    </TableCell>
                )}
            </TableRow>

            <PlayerContextMenu
                contextMenuPos={contextMenuPos}
                player={player}
                settings={settings}
                onClose={handleMenuClose}
            />

            <PlayerHoverInfo player={player} hoverMenuPos={hoverMenuPos} />
        </Fragment>
    );
};
interface PlayerHoverInfoProps {
    player: Player;
    hoverMenuPos: NullablePosition;
}

const makeInfoRow = (key: string, value: string): JSX.Element[] => {
    return [
        <Grid xs={3} key={`${key}-key`} padding={0}>
            <Typography variant={'button'} textAlign={'right'}>
                {key}
            </Typography>
        </Grid>,
        <Grid xs={9} key={`${key}-val`} padding={0}>
            <Typography variant={'body1'}>{value}</Typography>
        </Grid>
    ];
};

const PlayerHoverInfo = ({ player, hoverMenuPos }: PlayerHoverInfoProps) => {
    const { t } = useTranslation();

    return (
        <Popover
            open={hoverMenuPos !== null}
            sx={{
                pointerEvents: 'none'
            }}
            anchorReference="anchorPosition"
            anchorPosition={
                hoverMenuPos !== null
                    ? {
                          top: hoverMenuPos.mouseY,
                          left: hoverMenuPos.mouseX
                      }
                    : undefined
            }
            disablePortal={false}
            //disableRestoreFocus
        >
            <Paper style={{ maxWidth: 650 }} sx={{ padding: 1 }}>
                <Grid container spacing={1}>
                    <Grid xs={'auto'}>
                        <img
                            height={184}
                            width={184}
                            alt={player.name}
                            src={avatarURL(player.avatar_hash)}
                        />
                    </Grid>
                    <Grid xs>
                        <div>
                            <Grid container>
                                {...makeInfoRow(
                                    t('player_table.details.uid_label'),
                                    player.user_id.toString()
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.name_label'),
                                    player.name
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.visibility_label'),
                                    visibilityString(player.visibility)
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.vac_bans_label'),
                                    player.number_of_vac_bans.toString()
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.game_bans_label'),
                                    player.number_of_game_bans.toString()
                                )}
                            </Grid>
                        </div>
                    </Grid>
                    {player.notes.length > 0 && (
                        <Grid xs={12}>
                            <TextareaAutosize value={player.notes} />
                        </Grid>
                    )}
                    {player.matches && player.matches.length > 0 && (
                        <Grid xs={12}>
                            <TableContainer>
                                <Table size={'small'}>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell padding={'normal'}>
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.matches.origin_label'
                                                    }
                                                />
                                            </TableCell>

                                            <TableCell padding={'normal'}>
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.matches.type_label'
                                                    }
                                                />
                                            </TableCell>
                                            <TableCell
                                                padding={'normal'}
                                                width={'100%'}
                                            >
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.matches.tags_label'
                                                    }
                                                />
                                            </TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {player.matches?.map((match) => {
                                            return (
                                                <TableRow
                                                    key={`match-${match.origin}`}
                                                >
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.origin}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.matcher_type}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {match.attributes.join(
                                                                ', '
                                                            )}
                                                        </Typography>
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                    </TableBody>
                                </Table>
                            </TableContainer>
                        </Grid>
                    )}
                    {player.sourcebans && player.sourcebans.length > 0 && (
                        <Grid xs={12}>
                            <TableContainer>
                                <Table size={'small'}>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell padding={'normal'}>
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.sourcebans.site_name_label'
                                                    }
                                                />
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.sourcebans.created_label'
                                                    }
                                                />
                                            </TableCell>
                                            <TableCell padding={'normal'}>
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.sourcebans.perm_label'
                                                    }
                                                />
                                            </TableCell>
                                            <TableCell
                                                padding={'normal'}
                                                width={'100%'}
                                            >
                                                <Trans
                                                    i18nKey={
                                                        'player_table.details.sourcebans.reason_label'
                                                    }
                                                />
                                            </TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {player.sourcebans.map((ban) => {
                                            return (
                                                <TableRow
                                                    key={`sb-${ban.ban_id}`}
                                                >
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {ban.site_name}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {format(
                                                                parseJSON(
                                                                    ban.created_on
                                                                ),
                                                                'MM/dd/yyyy'
                                                            )}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'button'}
                                                        >
                                                            {ban.permanent
                                                                ? t('yes')
                                                                : t('no')}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            padding={1}
                                                            variant={'body1'}
                                                        >
                                                            {ban.reason}
                                                        </Typography>
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                    </TableBody>
                                </Table>
                            </TableContainer>
                        </Grid>
                    )}
                </Grid>
            </Paper>
        </Popover>
    );
};
