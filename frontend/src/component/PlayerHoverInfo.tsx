import { Trans, useTranslation } from 'react-i18next';
import Popover from '@mui/material/Popover';
import Paper from '@mui/material/Paper';
import Grid from '@mui/material/Unstable_Grid2';
import { avatarURL, Player, visibilityString } from '../api';
import { TextareaAutosize } from '@mui/material';
import TableContainer from '@mui/material/TableContainer';
import Table from '@mui/material/Table';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import TableBody from '@mui/material/TableBody';
import Typography from '@mui/material/Typography';
import { format, parseJSON } from 'date-fns';
import { NullablePosition } from './menu/common';
import React from 'react';

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

export const PlayerHoverInfo = ({
    player,
    hoverMenuPos
}: PlayerHoverInfoProps) => {
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
                            alt={player.personaname}
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
                                    player.personaname
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.visibility_label'),
                                    visibilityString(player.visibility)
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.vac_bans_label'),
                                    player.vac_bans.toString()
                                )}
                                {...makeInfoRow(
                                    t('player_table.details.game_bans_label'),
                                    player.game_bans.toString()
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
