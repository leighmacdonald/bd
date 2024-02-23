import { Button } from '@mui/material';
import SaveIcon from '@mui/icons-material/Save';
import { Trans } from 'react-i18next';
import { onClickProps } from '../util.ts';

export const SaveButton = ({
    onClick,
    disabled = false
}: onClickProps & { disabled?: boolean }) => {
    return (
        <Button
            startIcon={<SaveIcon />}
            color={'success'}
            variant={'contained'}
            onClick={onClick}
            disabled={disabled}
        >
            <Trans i18nKey={'button.save'} />
        </Button>
    );
};

export default SaveButton;
