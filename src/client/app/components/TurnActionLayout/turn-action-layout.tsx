import "./turn-action-layout.css";

type Props = {
    isShow: boolean;
}

export const TurnActionLayout: React.FC<Props> = ({isShow}) => {
    if(!isShow) return false

    return (
        <div className="layout">
            <div className="layout_capture">
                !180
            </div>
        </div>
    );
}

