import { useTranslation } from 'react-i18next';
import { IconMenuItem, NestedMenuItem } from 'mui-nested-menu';
import ArrowRightOutlinedIcon from '@mui/icons-material/ArrowRightOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import { logError, writeToClipboard } from '../../util';
import { SteamIDProps, SubMenuProps } from './common';
import * as SteamID from 'steamid';

export const SteamIDMenu = ({
    contextMenuPos,
    steam_id,
    onClose
}: SubMenuProps & SteamIDProps) => {
    const { t } = useTranslation();
    const id = new SteamID(steam_id);
    return (
        <NestedMenuItem
            rightIcon={<ArrowRightOutlinedIcon />}
            leftIcon={<ContentCopyOutlinedIcon color={'primary'} />}
            label={t('player_table.menu.copy_label')}
            parentMenuOpen={contextMenuPos !== null}
        >
            {[
                id.getSteam2RenderedID(),
                id.getSteam3RenderedID(),
                id.getSteamID64()
            ].map((sid) => {
                return (
                    <IconMenuItem
                        key={`steam-id-link-${sid}`}
                        onClick={async () => {
                            try {
                                await writeToClipboard(sid);
                            } catch (e) {
                                logError(e);
                            } finally {
                                onClose();
                            }
                        }}
                        label={sid}
                    />
                );
            })}
        </NestedMenuItem>
    );
};
