import { useEffect, useState } from "react";
import {TurnActionLayout} from "~/components/TurnActionLayout/turn-action-layout"
import {StunActionLayout} from "~/components/StunActionLayout/stun-action-layout"

import "./obs-overlay.css";

export function ObsOverlay() {
  const [layoutVisible, setLayoutVisible] = useState({
    turn: {
      visible: false,
      duration: 0,
    },
    stun: {
      visible: false,
      duration: 0,
    }
  })

  useEffect(() => {
    console.log('subscribe to event')
    const evs = new EventSource('http://localhost:8081/events');

    evs.onmessage = function(e) {
      const data = JSON.parse(e.data) as {type: 'turn' | 'stun', duration: number};
      console.log('new source event: ', data);

      setLayoutVisible({
        ...layoutVisible,
        [data.type]: {
          visible: true,
          duration: data.duration
        }
      })

      setTimeout(() => {
        setLayoutVisible({
          ...layoutVisible,
          [data.type]: {
            visible: false,
            duration: data.duration
          }
        })
      }, data.duration)
    };
    
    return () => {
      evs.close()
    }
  }, [layoutVisible])

  return (
    <div>
      obs-overlay
      <TurnActionLayout isShow={layoutVisible['turn'].visible} />
      <StunActionLayout isShow={layoutVisible['stun'].visible} duration={layoutVisible['stun'].duration} />
    </div>
  );
}

/*
const evs = new EventSource('/events');
evs.onmessage = function(e) {
    const d = JSON.parse(e.data);
    if (d.type === 'stun') {
        document.getElementById('stun').classList.toggle('active', d.active);
    } else if (d.type === 'turn') {
        const el = document.getElementById('turn');
        el.classList.remove('flash');
        void el.offsetWidth;
        el.classList.add('flash');
    }
};
*/