import { Button } from '@mui/material';
import RestartAltIcon from '@mui/icons-material/RestartAlt';
import { Trans } from 'react-i18next';
import { onClickProps } from '../util.ts';

export const ResetButton = ({ onClick }: onClickProps) => {
    return (
        <Button
            onClick={onClick}
            startIcon={<RestartAltIcon />}
            color={'warning'}
            variant={'contained'}
        >
            <Trans i18nKey={'button.reset'} />
        </Button>
    );
};

export default ResetButton;
