import "./stun-action-layout.css";

import {Timer} from '~/components/Timer/timer'

type Props = {
    isShow: boolean;
    duration: number
}

export const StunActionLayout: React.FC<Props> = ({isShow, duration}) => {
    if(!isShow) return false

    const offsetTimer =  (duration - 3000)/1000

    return (
        <div className="layout">
            <div className="layout-timer">
                <Timer numbers={[3,2,1]} offset={offsetTimer} />
            </div>
            <div className="layout-capture">
                <div className="layout-capture_capture">
                    STUNNED
                </div>
            </div>
        </div>
    );
}

