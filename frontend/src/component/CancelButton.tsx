import { Trans } from 'react-i18next';
import { Button } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import { onClickProps } from '../util.ts';

export const CancelButton = ({ onClick }: onClickProps) => {
    return (
        <Button
            startIcon={<CloseIcon />}
            color={'error'}
            variant={'contained'}
            onClick={onClick}
        >
            <Trans i18nKey={'button.cancel'} />
        </Button>
    );
};

export default CancelButton;
