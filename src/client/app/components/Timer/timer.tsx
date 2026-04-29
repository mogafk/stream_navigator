import "./timer.css";

type Props = {
  offset: number;
  numbers: number[];
}

export const Timer: React.FC<Props> = ({offset, numbers}) => {

  return (
    <div>
      {
        numbers.map(
          (number, index) => 
            <div
              className="number"
              style={{"animationDelay": `${offset+index}s`}}
            >{number}</div>
        )
      }
    </div>
  );
}

