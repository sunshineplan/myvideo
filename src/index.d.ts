type Category = {
  dongman: string;
  dianying: string;
  dianshiju: string;
  zongyi: string;
}

type List<T> = {
  [key in keyof T]: video[];
};

interface video {
  name: string;
  url: string;
  image: string;
  playlist: { [key: string]: play[] };
}

interface play {
  ep: string;
  m3u8: string;
}
