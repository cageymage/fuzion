import { Link, useParams } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { useLoginUrl } from '../../hooks/useLoginUrl'
import { formatDeadline } from '../../lib/format'
import styles from './Survey.module.css'
import { SurveyForm } from './SurveyForm'
import { SurveyResults } from './SurveyResults'
import { useSurvey } from './useSurvey'

export function Survey() {
  const { id = '' } = useParams()
  const survey = useSurvey(id)
  const currentUser = useCurrentUser()
  const loginUrl = useLoginUrl()

  if (survey.isPending) {
    return (
      <section className={styles.page}>
        <p className={styles.notice}>Loading survey…</p>
      </section>
    )
  }
  if (survey.isError) {
    const notFound = survey.error instanceof ApiError && survey.error.status === 404
    return (
      <section className={styles.page}>
        <h1 className={styles.title}>{notFound ? 'Survey not found' : 'Survey'}</h1>
        {!notFound && (
          <p role="alert" className={styles.error}>
            The survey could not be loaded.
          </p>
        )}
        <Link to="/surveys" className={styles.backLink}>
          All surveys
        </Link>
      </section>
    )
  }

  const { data } = survey
  const closed = data.status === 'closed'
  return (
    <section className={styles.page}>
      <Link to="/surveys" className={styles.backLink}>
        All surveys
      </Link>
      <h1 className={styles.title}>{data.title}</h1>
      <p className={styles.deadline}>
        {closed ? 'Closed' : 'Closes'} {formatDeadline(data.closesAt)}
      </p>
      {data.description && <p className={styles.description}>{data.description}</p>}
      {closed ? (
        <SurveyResults surveyId={data.id} />
      ) : currentUser.isPending ? null : currentUser.data?.isGuildMember ? (
        <SurveyForm survey={data} />
      ) : currentUser.data ? (
        <p className={styles.notice}>
          Only members of the guild Discord server can respond. If you have just joined, log out
          and back in to refresh your membership.
        </p>
      ) : (
        <p>
          <a href={loginUrl}>Log in with Discord</a> to respond.
        </p>
      )}
    </section>
  )
}
